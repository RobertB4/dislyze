package auth

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"

	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"

	"github.com/crewjam/saml"
)

func (h *AuthHandler) SSOMetadata(w http.ResponseWriter, r *http.Request) {
	metadata, err := h.generateSPMetadata()
	if err != nil {
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write(metadata)
}

func (h *AuthHandler) generateSPMetadata() ([]byte, error) {
	keyBlock, _ := pem.Decode([]byte(h.env.SAMLServiceProviderPrivateKey))
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode SP private key")
	}
	privateKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SP private key: %w", err)
	}

	certBlock, _ := pem.Decode([]byte(h.env.SAMLServiceProviderCertificate))
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode SP certificate")
	}
	spCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SP certificate: %w", err)
	}

	acsURL, _ := url.Parse(h.env.FrontendURL + "/api/auth/sso/acs")
	spMetadataURL, _ := url.Parse(h.env.FrontendURL + "/api/auth/sso/metadata")

	sp := &saml.ServiceProvider{
		Key:               privateKey.(crypto.Signer),
		Certificate:       spCert,
		MetadataURL:       *spMetadataURL,
		AcsURL:            *acsURL,
		EntityID:          h.env.FrontendURL,
		AuthnNameIDFormat: saml.EmailAddressNameIDFormat,
		SignatureMethod:   "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
	}

	metadata, err := xml.MarshalIndent(sp.Metadata(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return metadata, nil
}
