/*
HTTP Parameter Pollution (HPP) is a web application attack where attackers exploit how web applications handle multiple parameters with the same name. By injecting encoded delimiters in the URL, they can potentially manipulate or retrieve hidden information, bypass security measures, or even redirect users to malicious websites.
Here's a more detailed explanation:
How it works:
HPP involves crafting HTTP requests with multiple parameters having the same name. The web application's parsing logic might not handle these duplicate parameters correctly, leading to unexpected behavior.
Exploitation:
Attackers can leverage this to:
Bypass input validation: By manipulating parameters, they can potentially bypass filters designed to prevent malicious input.
Alter application behavior: HPP can change how a web page or application functions, leading to unexpected redirects or modifications to data.
Escalate privileges: In some cases, HPP can be used to gain unauthorized access or escalate privileges within the application.
Cross-site Scripting (XSS): HPP can sometimes be used to inject malicious scripts into a website's output, leading to XSS attacks.
Example:
An attacker might craft a URL with multiple "category" parameters. If the application only processes the first "category" parameter and ignores the rest, the attacker could potentially inject a malicious "category" value that gets processed later.
*/

package middlewares

import (
	"fmt"
	"net/http"
	"strings"
)

type HPPOptions struct {
	CheckQuery 					bool
	CheckBody 					bool
	CheckBodyOnlyForContentType string
	WhiteList 					[]string
}

// This is a higher‑order function returning a closure — a very common Go idiom for building middleware.
func Hpp(options HPPOptions) func (http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if options.CheckBody && r.Method == http.MethodPost && isCorrectContentType(r, options.CheckBodyOnlyForContentType){
				filterBodyParams(r, options.WhiteList)
			}
			if options.CheckQuery && r.URL.Query() != nil {
				filterQueryParams(r, options.WhiteList)
			}
			next.ServeHTTP(w, r)
		})
	}
}


func isCorrectContentType(r *http.Request, contentType string) bool {
 return strings.Contains(r.Header.Get("Content-Type"), contentType)
}


func filterBodyParams(r *http.Request, whiteList []string) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		return
	}

	for k, v := range r.Form {
		if len(v) > 1 {
			r.Form.Set(k, v[0])
		}
		if !isWhiteListed(k, whiteList) {
			delete(r.Form, k)
		}
	}
}

func filterQueryParams(r *http.Request, whiteList []string) {
	query := r.URL.Query()


	for k, v := range query {
		if len(v) > 1 {
			query.Set(k, v[0])
		}
		if !isWhiteListed(k, whiteList) {
			query.Del(k)
		}
	}
	r.URL.RawQuery = query.Encode()
}

func isWhiteListed(param string, whitelist []string) bool {
	for _, v := range whitelist {
		if param == v {
			return true
		}
	}
	return false
}