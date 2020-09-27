package paymail

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
)

/*
Default Response:
{
  "bsvalias": "1.0",
  "handle": "<alias>@<domain>.<tld>",
  "pubkey": "..."
}
*/

// PKI is the result returned
type PKI struct {
	StandardResponse
	BsvAlias string `json:"bsvalias"` // Version of Paymail
	Handle   string `json:"handle"`   // The <alias>@<domain>.<tld>
	PubKey   string `json:"pubkey"`   // The related PubKey
}

// GetPKI will return a valid PKI response for a given alias@domain.tld
//
// Specs: http://bsvalias.org/03-public-key-infrastructure.html
func (c *Client) GetPKI(pkiURL, alias, domain string) (response *PKI, err error) {

	// Require a valid url
	if len(pkiURL) == 0 || !strings.Contains(pkiURL, "https://") {
		err = fmt.Errorf("invalid url: %s", pkiURL)
		return
	}

	// Set the base url and path, assuming the url is from the prior GetCapabilities() request
	// https://<host-discovery-target>/{alias}@{domain.tld}/id
	reqURL := strings.Replace(strings.Replace(pkiURL, "{alias}", alias, -1), "{domain.tld}", domain, -1)

	// Set the user agent
	req := c.Resty.R().SetHeader("User-Agent", c.Options.UserAgent)

	// Enable tracing
	if c.Options.RequestTracing {
		req.EnableTrace()
	}

	// Fire the request
	var resp *resty.Response
	if resp, err = req.Get(reqURL); err != nil {
		return
	}

	// Start new response
	response = new(PKI)

	// Tracing enabled?
	if c.Options.RequestTracing {
		response.Tracing = resp.Request.TraceInfo()
	}

	// Test the status code (200 or 304 is valid)
	response.StatusCode = resp.StatusCode()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotModified {
		je := &JSONError{}
		if err = json.Unmarshal(resp.Body(), je); err != nil {
			return
		}
		err = fmt.Errorf("bad response from paymail provider: code %d, message: %s", response.StatusCode, je.Message)
		return
	}

	// Decode the body of the response
	if err = json.Unmarshal(resp.Body(), &response); err != nil {
		return
	}

	// Invalid version detected
	if len(response.BsvAlias) == 0 {
		err = fmt.Errorf("missing bsvalias version")
		return
	}

	// Check basic requirements (handle should match our alias@domain.tld)
	if response.Handle != alias+"@"+domain {
		err = fmt.Errorf("pki response handle %s does not match paymail address: %s", response.Handle, alias+"@"+domain)
		return
	}

	// Check the PubKey length
	if len(response.PubKey) == 0 {
		err = fmt.Errorf("pki response is missing a PubKey value")
	} else if len(response.PubKey) != PubKeyLength {
		err = fmt.Errorf("returned pubkey is not the required length of %d, got: %d", PubKeyLength, len(response.PubKey))
	}

	return
}
