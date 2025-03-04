package dnssvc

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/AdguardTeam/AdGuardDNS/internal/agd"
	"github.com/AdguardTeam/AdGuardDNS/internal/agdnet"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnsserver"
	"github.com/AdguardTeam/AdGuardDNS/internal/errcoll"
	"github.com/AdguardTeam/AdGuardDNS/internal/optlog"
	"github.com/AdguardTeam/golibs/errors"
)

// Device ID Extraction

// deviceIDFromClientServerName extracts and validates a device ID.  cliSrvName
// is the server name as sent by the client.  wildcards are the domain wildcards
// for device ID detection.
func deviceIDFromClientServerName(
	cliSrvName string,
	wildcards []string,
) (id agd.DeviceID, err error) {
	if cliSrvName == "" {
		// No server name in ClientHello, so the request is probably made on the
		// IP address.
		return "", nil
	}

	matchedDomain := ""
	for _, wildcard := range wildcards {
		// Assume that wildcards have been validated for this prefix in the
		// configuration parsing.
		domain := wildcard[len("*."):]
		matched := agdnet.IsImmediateSubdomain(cliSrvName, domain)
		if matched {
			matchedDomain = domain

			break
		}
	}

	if matchedDomain == "" {
		return "", nil
	}

	optlog.Debug2("device id check: matched %q from %q", matchedDomain, wildcards)

	idStr := cliSrvName[:len(cliSrvName)-len(matchedDomain)-1]
	id, err = agd.NewDeviceID(idStr)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return "", err
	}

	return id, nil
}

// deviceIDFromDoHURL extracts the device ID from the path of the client's
// DNS-over-HTTPS request.
func deviceIDFromDoHURL(u *url.URL) (id agd.DeviceID, err error) {
	origPath := u.Path
	parts := strings.Split(path.Clean(origPath), "/")
	if parts[0] == "" {
		parts = parts[1:]
	}

	if parts[0] == "" ||
		!strings.HasSuffix(dnsserver.PathDoH, parts[0]) &&
			!strings.HasSuffix(dnsserver.PathJSON, parts[0]) {
		return "", fmt.Errorf("bad path %q", u.Path)
	}

	switch len(parts) {
	case 1:
		// Just /dns-query, no device ID.
		return "", nil
	case 2:
		id, err = agd.NewDeviceID(parts[1])
		if err != nil {
			// Don't wrap the error, because it's informative enough as is.
			return "", err
		}
	default:
		return "", fmt.Errorf("bad path %q: extra parts", u.Path)
	}

	return id, nil
}

// deviceIDError is an error about a bad device ID or other issues found during
// device ID checking.
type deviceIDError struct {
	err error
	typ string
}

// type check
var _ error = (*deviceIDError)(nil)

// Error implements the error interface for *deviceIDError.
func (err *deviceIDError) Error() (msg string) {
	return fmt.Sprintf("%s device id check: %s", err.typ, err.err)
}

// type check
var _ errors.Wrapper = (*deviceIDError)(nil)

// Unwrap implements the errors.Wrapper interface for *deviceIDError.
func (err *deviceIDError) Unwrap() (unwrapped error) { return err.err }

// type check
var _ errcoll.RavenReportableError = (*deviceIDError)(nil)

// IsRavenReportable implements the errcoll.RavenReportableError interface for
// *deviceIDError.
func (err *deviceIDError) IsRavenReportable() (ok bool) { return false }

// deviceIDFromContext extracts the device from the server name of the TLS
// client's DoH, DoT, or DoQ request, using the provided domain name wildcards,
// and also from the DoH request, using the path of the HTTP URL.  If the
// protocol is not one of these, id is an empty string and err is nil.
//
// Any returned errors will have the underlying type of *deviceIDError.
func deviceIDFromContext(
	ctx context.Context,
	proto agd.Protocol,
	wildcards []string,
) (id agd.DeviceID, err error) {
	ci := dnsserver.MustClientInfoFromContext(ctx)

	if proto == agd.ProtoDoH {
		id, err = deviceIDFromDoHURL(ci.URL)
		if err != nil {
			return "", &deviceIDError{
				err: err,
				typ: "http url",
			}
		} else if id != "" {
			return id, nil
		}

		// Go on and check the domain name as well.
	} else if proto != agd.ProtoDoT && proto != agd.ProtoDoQ {
		return "", nil
	}

	if len(wildcards) == 0 {
		return "", nil
	}

	cliSrvName := ci.TLSServerName
	id, err = deviceIDFromClientServerName(cliSrvName, wildcards)
	if err != nil {
		return "", &deviceIDError{
			err: err,
			typ: "tls server name",
		}
	}

	return id, nil
}
