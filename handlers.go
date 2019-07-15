package main

import "net/http"

func GetHandler() (http.HandlerFunc) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})
}

func PostHandler() (http.HandlerFunc) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}
func DeleteHandler() (http.HandlerFunc) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})
}
