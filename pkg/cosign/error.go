package cosign

import (
	"errors"

	cosignError "github.com/sigstore/cosign/v2/cmd/cosign/errors"
)

// isNoMatchingSignatureErr checks if the error is of type ErrNoMatchingSignature
func isNoMatchingSignatureErr(err error) bool {
	var ce *cosignError.CosignError
	if errors.As(err, &ce) && ce.Code == cosignError.NoMatchingSignature {
		return true
	}
	return false
}

// isImageWithoutSignatureErr checks if the error is of type ErrNoSignaturesFound
func isImageWithoutSignatureErr(err error) bool {
	var ce *cosignError.CosignError
	if errors.As(err, &ce) && ce.Code == cosignError.ImageWithoutSignature {
		return true
	}
	return false
}

// isNoCertificateFoundOnSignatureErr checks if the error is of type ErrNoCertificateFoundOnSignature
func isNoCertificateFoundOnSignatureErr(err error) bool {
	var ce *cosignError.CosignError
	if errors.As(err, &ce) && ce.Code == cosignError.NoCertificateFoundOnSignature {
		return true
	}
	return false
}
