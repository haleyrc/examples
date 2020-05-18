// Package versioning contains an experiment for supporting old request versions
// using path variables and custom unmarshaling behavior.
//
// Scenario
//
// We provide a greeter service with a single endpoint. A client sends a name in
// a JSON request and the service responds with a greeting in a JSON response.
//
// Our initial version accepted a first and last name, but after some production
// use, it was determined that this was too restrictive for users with
// non-English names. As a result, we created a new request shape that accepts
// a single name parameter, letting clients determine how it should be
// formatted.
//
// Since we can't force existing clients to update, we need to continue to
// support the original request version. In fairly standard fashion, we decided
// to split into two URLs:
//
//   /v1/greet
//   /v2/greet
//
// Since the underlying business logic is the same, however, we did not want to
// deal with an explosion of handlers just to handle multiple request shapes.
//
// Solution
//
// In order to deprecate the original version but continue support, we renamed
// the original request type from GreetRequest to GreetRequestV1. We also
// created a new GreetRequest type with the new shape. GreetRequest will always
// be the most current version, with older version being renamed to make their
// deprecated status more obvious.
//
// We then embedded a pointer to the new GreetRequest type inside of the
// GreetRequestV1 type. The reason for this will become obvious soon.
//
// With the new types in place, we updated the GreetHandler to create the
// greeting using the single Name field of GreetRequest. At this point, we are
// properly supporting the new version, but older clients would either need a
// separate handler (not ideal), or our existing handler would need to be able
// to handle multiple request shapes. Instead of shoving all of that logic into
// the controller, however, we opted to add custom unmarshaling behavior to our
// request types. We accomplish this in a couple ways.
//
// For our deprecated GreetRequestV1 type, we add an UnmarshalJSON method that
// implements json.Unmarshaler by aliasing the type and calling json.Unmarshal
// directly. We then take the separate first and last name fields and combine
// them into the single name field on the embedded GreetRequest.
//
// For our new type, we add a few new methods. The first is a helper for
// wrapping a GreetRequest with a GreetRequestV1. This is then used in the
// second new method, Decode, when decoding a version 1 request shape. By
// embedding our GreetRequest, GreetRequestV1.UnmarshalJSON can set fields
// directly, transparently to the user. In the case we are looking at a version
// 2 shape (indicated by the version path parameter), we can simply decode
// directly.
//
// Finally, the handler is updated to call Decode directly on a GreetRequest
// while passing in the version pulled from the path. Now we can support both
// endpoints using a single handler. If the API needs to be updated again, we
// simply perform the same process:
//
//   - Rename the deprecated version
//   - Embed the GreetRequest pointer
//   - Add a custom UnmarshalJSON to the deprecated version
//   - Create the new type (note that now we'll need helpers for all supported
//     version)
//   - Add cases to the version switch for all supported versions using the new
//     helpers
//
// Everything else from this point should work as intended and we can see
// exactly how we're converting from one type to another without magic.
package versioning

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// GreetRequestV1 represents a previous version of the request shape we want to
// support. In this version, we allowed clients to supply both a first and last
// name.
type GreetRequestV1 struct {
	*GreetRequest

	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func (r GreetRequestV1) UnmarshalJSON(b []byte) error {
	type grv1 GreetRequestV1
	var v1r grv1
	if err := json.Unmarshal(b, &v1r); err != nil {
		return err
	}
	r.GreetRequest.Name = v1r.FirstName + " " + v1r.LastName
	return nil
}

type GreetRequest struct {
	Name string `json:"name"`
}

func (gr *GreetRequest) fromV1() json.Unmarshaler {
	return &GreetRequestV1{GreetRequest: gr}
}

func (gr *GreetRequest) Decode(version string, r io.Reader) error {
	dec := json.NewDecoder(r)

	var err error
	switch version {
	case "v1":
		err = dec.Decode(gr.fromV1())
	default:
		err = dec.Decode(gr)
	}

	return err
}

func GreetHandler(w http.ResponseWriter, r *http.Request) {
	version := mux.Vars(r)["version"]

	var req GreetRequest
	if err := req.Decode(version, r.Body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	greeting := fmt.Sprintf("Hello %s!", req.Name)
	json.NewEncoder(w).Encode(map[string]string{"greeting": greeting})
}

type App struct{}

func (a *App) Run() error {
	router := mux.NewRouter()
	router.HandleFunc("/{version}/greet", GreetHandler)
	return http.ListenAndServe(":8080", router)
}
