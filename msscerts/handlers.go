package msscerts

import (
	"github.com/pkg/errors"
	"go_report/failure"
	"net/http"
)

const CertCtxVar = "mssCertificate"

type TokenRequest struct {
	Certificate string `json:"appCert"`
}

func (man *Manager) AddCertificateHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cert := r.Context().Value(string(CertCtxVar)).(string)
		if cert == "" {
			failure.Fail(w, failure.New(errors.New("No certificate found in context"), http.StatusBadRequest, "no certificate provided"))
			return
		}
		if err := man.AddCertificate(cert); err != nil {
			failure.Fail(w, failure.New(err, http.StatusInternalServerError, "could not add certificate"))
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
}

func (man *Manager) RemoveCertificateHandler() http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cert := r.Context().Value(string(CertCtxVar)).(string)
		if cert == "" {
			failure.Fail(w, failure.New(errors.New("No certificate found in context"), http.StatusBadRequest, "no certificate provided"))
			return
		}
		if err  := man.RemoveCertificate(cert); err != nil {
			failure.Fail(w, failure.New(err, http.StatusInternalServerError, "could not remove certificate"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}