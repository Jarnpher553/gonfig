package route

type Router struct {
	URL    string
	Method string
}

func NewRouter(method, url string) *Router {
	return &Router{
		URL:    url,
		Method: method,
	}
}
