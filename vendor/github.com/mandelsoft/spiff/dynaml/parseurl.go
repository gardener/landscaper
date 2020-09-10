package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"

	"net/url"
	"strconv"
)

const F_ParseURL = "parseurl"

func init() {
	RegisterFunction(F_ParseURL, func_parseurl)
}

func func_parseurl(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var err error
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("invalid argument count for %s", F_ParseURL)
	}

	urlstr, ok := arguments[0].(string)
	if !ok {
		return info.Error("argument for %s must be a string", F_ParseURL)
	}

	u, err := url.Parse(urlstr)
	if err != nil {
		return info.Error("invalid argument for %s: %s", F_ParseURL, err)
	}

	result := map[string]yaml.Node{}

	result["scheme"] = NewNode(u.Scheme, binding)
	result["host"] = NewNode(u.Hostname(), binding)
	result["path"] = NewNode(u.Path, binding)
	result["fragment"] = NewNode(u.Fragment, binding)

	if u.Port() != "" {
		port, err := strconv.ParseInt(u.Port(), 10, 64)
		if err == nil {
			result["port"] = NewNode(port, binding)
		}
	}
	result["query"] = NewNode(u.RawQuery, binding)

	query := map[string]yaml.Node{}
	for k, v := range u.Query() {
		p := []yaml.Node{}
		for _, a := range v {
			p = append(p, NewNode(a, binding))
		}
		query[k] = NewNode(p, binding)
	}
	result["values"] = NewNode(query, binding)

	if u.User != nil {
		p := map[string]yaml.Node{}
		p["username"] = NewNode(u.User.Username(), binding)
		if pass, ok := u.User.Password(); ok {
			p["password"] = NewNode(pass, binding)
		}
		result["userinfo"] = NewNode(p, binding)
	}

	return result, info, true
}
