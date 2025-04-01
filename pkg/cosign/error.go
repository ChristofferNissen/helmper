package cosign

import "strings"

// // isNoMatchingSignatureErr checks if the error is of type ErrNoMatchingSignature
//
//	func isNoMatchingSignatureErr(err error) bool {
//		var ce *cosignError.CosignError
//		if errors.As(err, &ce) && ce.Code == cosignError.NoMatchingSignature {
//			return true
//		}
//		return false
//	}
//
// // isImageWithoutSignatureErr checks if the error is of type ErrNoSignaturesFound
//
//	func isImageWithoutSignatureErr(err error) bool {
//		var ce *cosignError.CosignError
//		if errors.As(err, &ce) && ce.Code == cosignError.ImageWithoutSignature {
//			return true
//		}
//		return false
//	}
//
// // isNoCertificateFoundOnSignatureErr checks if the error is of type ErrNoCertificateFoundOnSignature
//
//	func isNoCertificateFoundOnSignatureErr(err error) bool {
//		var ce *cosignError.CosignError
//		if errors.As(err, &ce) && ce.Code == cosignError.NoCertificateFoundOnSignature {
//			return true
//		}
//		return false
//	}

func isNoCertificateFoundOnSignatureErr(err string) bool {
	return strings.Contains(err, "no certificate found on signature")
}

func isNoMatchingSignatureErr(err string) bool {
	return strings.Contains(err, "no matching signatures")
}

func isImageWithoutSignatureErr(err string) bool {
	return strings.Contains(err, "image without signature")
}
