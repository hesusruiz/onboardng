package server

import (
	"fmt"
	"net/http"
	"net/url"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

// QRCodeHandler is a HTTP handler that displays a QR code with the given URL.
// The URL is passed as a query parameter named "credential_offer_uri".
// It converts the "credentialOfferUri" to a full URL that can be used to start the wallet flow.
func (s *Server) QRCodeHandler(w http.ResponseWriter, r *http.Request) {

	// Get the credential_offer_uri from the parameter in the request URL
	credentialOfferURI := r.URL.Query().Get("credential_offer_uri")

	// Return error if credentialOfferURI is not provided
	if credentialOfferURI == "" {
		http.Error(w, "credential_offer_uri is required", http.StatusBadRequest)
		return
	}

	fmt.Printf("credentialOfferURI: %s\n", credentialOfferURI)

	//URLEncode the credentialOfferURI
	credentialOfferURI = url.QueryEscape(credentialOfferURI)

	// Add the prefix for the wallet credential issuance callback URL
	credentialOfferURI = s.WalletURL + "?credential_offer_uri=" + credentialOfferURI

	fmt.Printf("Displaying credentialOfferURI: %s\n", credentialOfferURI)

	n := QRCodePage("TODO", "DOME marketplace", credentialOfferURI)
	n.Render(w)
}

func QRCodePage(title string, appName string, url string) Node {

	qrScript := fmt.Sprintf(
		`
window.addEventListener('load', () => {
	const qrContainer = document.getElementById('qrcode');

	qrCodeInstance = new QRCode(qrContainer, {
		text: "%s",
		width: 180,
		height: 180,
		colorDark: "#070f24",
		colorLight: "#ffffff",
		correctLevel: QRCode.CorrectLevel.M
	});
});`,
		url,
	)

	props := HTML5Props{
		Language: "en",
		Title:    title,
		Head: []Node{
			Link(Attr("rel", "stylesheet"), Attr("href", "myassets/w3.css")),
			Link(Attr("rel", "stylesheet"), Attr("href", "myassets/dome.css")),
			Link(Attr("rel", "stylesheet"), Attr("href", "https://fonts.googleapis.com/css2?family=Blinker:wght@100;200;300;400;600;700;800;900&family=Caladea:ital,wght@0,400;0,700;1,400;1,700&display=swap")),
			Link(Attr("rel", "stylesheet"), Attr("href", "https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@20..48,100..700,0..1,-50..200&icon_names=check_circle,counter_1,counter_2,counter_3")),
			Script(Attr("defer", ""), Attr("src", "https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js")),
			Script(Attr("defer", ""), Attr("src", "https://cdnjs.cloudflare.com/ajax/libs/qrcodejs/1.0.0/qrcode.min.js")),
		},
		Body: []Node{
			QRCodeHeader(),
			Main(
				Div(Style("background-color: #EDF2FA; min-height: 100vh;"),
					Div(Class("w3-padding-32 dome-content"),
						Div(ID("qrcode"), Style("width: 180px; height: 180px;")),
					),
				),
			),
			Script(Raw(qrScript)),
		},
	}

	return HTML5(props)

}

func QRCodeHeader() Node {
	return Raw(`
<div class="dome-header">
    <div class="dome-content">
        <div class="w3-bar">
            <div class="w3-bar-item padding-right-0">
                <a href="#">
                    <img src="myassets/logos/DOME_Icon_White.svg" alt="DOME Icon" style="width:100%;max-height:32px">
                </a>
            </div>
            <div class="w3-bar-item">
                <span class="blinker-semibold w3-xlarge nowrap">DOME Marketplace</span>
            </div>
        </div>
    </div>
</div>
`)
}
