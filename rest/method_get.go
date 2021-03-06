package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rs/rest-layer/resource"

	"golang.org/x/net/context"
)

// listGet handles GET resquests on a resource URL
func listGet(ctx context.Context, r *http.Request, route *RouteMatch) (status int, headers http.Header, body interface{}) {
	page := 1
	perPage := 0
	if route.Method != "HEAD" {
		if l := route.Resource().Conf().PaginationDefaultLimit; l > 0 {
			perPage = l
		} else {
			// Default value on non HEAD request for perPage is -1 (pagination disabled)
			perPage = -1
		}
		if p := r.URL.Query().Get("page"); p != "" {
			i, err := strconv.ParseUint(p, 10, 32)
			if err != nil {
				return 422, nil, &Error{422, "Invalid `page` paramter", nil}
			}
			page = int(i)
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			i, err := strconv.ParseUint(l, 10, 32)
			if err != nil {
				return 422, nil, &Error{422, "Invalid `limit` paramter", nil}
			}
			perPage = int(i)
		}
		if perPage == -1 && page != 1 {
			return 422, nil, &Error{422, "Cannot use `page' parameter with no `limit' paramter on a resource with no default pagination size", nil}
		}
	}
	lookup, e := route.Lookup()
	if e != nil {
		return e.Code, nil, e
	}
	list, err := route.Resource().Find(ctx, lookup, page, perPage)
	if err != nil {
		e = NewError(err)
		return e.Code, nil, e
	}
	for _, item := range list.Items {
		item.Payload, err = lookup.ApplySelector(ctx, route.Resource(), item.Payload, func(path string) (*resource.Resource, error) {
			router, ok := IndexFromContext(ctx)
			if !ok {
				return nil, errors.New("router not available in context")
			}
			rsrc, _, found := router.GetResource(path, route.Resource())
			if !found {
				return nil, fmt.Errorf("invalid resource reference: %s", path)
			}
			return rsrc, err
		})
		if err != nil {
			e = NewError(err)
			return e.Code, nil, e
		}
	}
	return 200, nil, list
}
