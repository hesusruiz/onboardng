package server

import (
	"encoding/json"
	"time"

	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/pdf"
)

const templateHTML = `
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="generator" content="DOME Docs Visual Template Editor (HugeRTE)">
    <meta name="domedocs-meta"
        content="{&quot;title&quot;:&quot;DOME_CSP_Appointment&quot;,&quot;headerLeft&quot;:&quot;DOME CSP Account administrator appointment Form&quot;,&quot;headerRight&quot;:&quot;version March 2026&quot;,&quot;footerLeft&quot;:&quot;&quot;,&quot;showPageNumbers&quot;:true,&quot;fontFamily&quot;:&quot;sans&quot;,&quot;marginPreset&quot;:&quot;wide&quot;}">
    <title>DOME_CSP_Appointment</title>
    <style>
        @page {
            size: A4;
            margin: 30mm 28mm 30mm 28mm;

            @top-left {
                content: "DOME CSP Account administrator appointment Form";
                font-family: "Avenir Next", Avenir, Inter, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
                font-size: 9pt;
                font-style: italic;
                color: #555;
                vertical-align: bottom;
                padding-bottom: 6pt;
            }

            @top-right {
                content: "version March 2026";
                font-family: "Avenir Next", Avenir, Inter, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
                font-size: 9pt;
                font-style: italic;
                color: #555;
                vertical-align: bottom;
                padding-bottom: 6pt;
            }

            @bottom-left {
                content: "";
                font-family: "Avenir Next", Avenir, Inter, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
                font-size: 9pt;
                color: #555;
                vertical-align: top;
                padding-top: 6pt;
            }

            @bottom-right {
                content: "Page " counter(page) " out of " counter(pages);
                font-family: "Avenir Next", Avenir, Inter, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
                font-size: 9.5pt;
                color: #333;
                vertical-align: top;
                padding-top: 6pt;
            }
        }

        * {
            box-sizing: border-box;
        }

        html,
        body {
            margin: 0;
            padding: 0;
            background-color: #eef1f5;
            color: #1a1f24;
            font-family: "Avenir Next", Avenir, Inter, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            -webkit-print-color-adjust: exact;
            print-color-adjust: exact;
        }

        .legal-document,
        .document-sheet {
            width: 210mm;
            min-height: 297mm;
            margin: 16mm auto;
            padding: 30mm 28mm 30mm 28mm;
            background: #ffffff;
            box-shadow: 0 2px 14px rgba(15, 23, 42, 0.1);
            display: flex;
            flex-direction: column;
        }

        .doc-running-header {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 12pt;
            font-size: 9pt;
            font-style: italic;
            color: #555;
            margin-bottom: 14pt;
            padding-bottom: 4pt;
            border-bottom: 0.5pt solid #e2e8f0;
        }

        .document-body {
            flex: 1 1 auto;
        }

        .doc-footnotes {
            margin-top: 22pt;
            padding-top: 8pt;
            border-top: 0.5pt solid #777;
            font-size: 9pt;
            line-height: 1.4;
            color: #333;
        }

        .doc-footnotes ol.footnotes-list {
            margin: 0;
            padding-left: 16pt;
        }

        .doc-footnotes li {
            margin: 0 0 4pt 0;
        }

        .doc-footnotes a {
            color: #1155cc;
            text-decoration: underline;
        }

        .doc-running-footer {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 12pt;
            font-size: 9pt;
            color: #444;
            margin-top: 18pt;
            padding-top: 6pt;
            border-top: 0.5pt solid #e2e8f0;
        }

        .document-body {
            font-family: "Avenir Next", Avenir, Inter, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            font-size: 10.5pt;
            line-height: 1.48;
            color: #1a1f24;
            word-wrap: break-word;
        }

        .document-body>*:first-child {
            margin-top: 0;
        }

        .document-body p {
            margin: 0 0 8pt 0;
        }

        .document-body header.doc-header {
            margin: 0 0 12pt 0;
        }

        .document-body section.doc-section {
            margin: 12pt 0;
        }

        .document-body section.doc-subsection {
            margin: 8pt 0;
        }

        .document-body article.clause-article {
            margin: 10pt 0;
            padding-left: 8pt;
            border-left: 2pt solid #cbd5e1;
        }

        .document-body h1.doc-title {
            font-size: 12.5pt;
            font-weight: 700;
            color: #006666;
            text-align: center;
            margin: 0 0 12pt 0;
            line-height: 1.35;
        }

        .document-body h2.section-heading {
            font-size: 11.5pt;
            font-weight: 700;
            font-style: italic;
            color: #006666;
            border-bottom: 0.75pt solid #444;
            padding-bottom: 3pt;
            margin: 14pt 0 10pt 0;
            line-height: 1.35;
            break-after: avoid;
        }

        .document-body h2.section-heading-plain {
            font-size: 11.5pt;
            font-weight: 700;
            font-style: italic;
            color: #006666;
            margin: 14pt 0 8pt 0;
            line-height: 1.35;
            break-after: avoid;
        }

        .document-body h3 {
            font-size: 11pt;
            font-weight: 700;
            color: #1a1f24;
            margin: 11pt 0 6pt 0;
            line-height: 1.35;
            break-after: avoid;
        }

        .document-body h4 {
            font-size: 10.5pt;
            font-weight: 600;
            color: #334155;
            margin: 9pt 0 4pt 0;
            line-height: 1.35;
            break-after: avoid;
        }

        .document-body time {
            font: inherit;
            color: inherit;
        }

        .document-body ol,
        .document-body ul {
            margin: 4pt 0 10pt 0;
            padding-left: 20pt;
        }

        .document-body li {
            margin: 0 0 5pt 0;
            line-height: 1.48;
        }

        .document-body a {
            color: #1155cc;
            text-decoration: underline;
        }

        .document-body aside.callout-box,
        .document-body div.callout-box {
            border: 0.75pt solid #555;
            padding: 9pt 12pt;
            margin: 10pt 0 14pt 0;
            background-color: #fafcfd;
            font-size: 10pt;
            line-height: 1.42;
            break-inside: avoid;
        }

        .document-body aside.callout-box p,
        .document-body div.callout-box p {
            margin: 0 0 6pt 0;
        }

        .document-body aside.callout-box p:last-child,
        .document-body div.callout-box p:last-child {
            margin-bottom: 0;
        }

        .document-body blockquote.legal-quote {
            margin: 10pt 0 10pt 12pt;
            padding: 6pt 12pt;
            border-left: 2.5pt solid #006666;
            background-color: #f8fafc;
            font-style: italic;
        }

        .document-body blockquote.legal-quote p:last-child {
            margin-bottom: 0;
        }

        .document-body footer.signature-section {
            margin-top: 14pt;
            padding-top: 10pt;
            border-top: 0.75pt solid #444;
            break-inside: avoid;
        }

        .document-body hr {
            border: none;
            border-top: 0.75pt solid #444;
            margin: 14pt 0;
        }

        .document-body hr.page-break,
        .document-body div.page-break {
            break-before: page;
            page-break-before: always;
            height: 0;
            margin: 0;
            border: none;
        }

        .document-body table {
            width: 100%;
            border-collapse: collapse;
            margin: 8pt 0 12pt 0;
            table-layout: fixed;
            break-inside: avoid;
        }

        .document-body table.form-table td,
        .document-body table.form-table th {
            border: 0.6pt solid #444;
            padding: 5pt 8pt;
            vertical-align: top;
            text-align: left;
            font-size: 10.25pt;
        }

        .document-body table.form-table td.field-code {
            width: 7%;
            font-size: 8.5pt;
            font-weight: 500;
            color: #64748b;
            background-color: #f4f7f9;
            text-align: center;
        }

        .document-body table.form-table th[scope="row"] {
            width: 35%;
            font-weight: 500;
            background-color: #fbfcfd;
            text-align: left;
        }

        .document-body table.form-table td.field-value {
            width: 58%;
            background-color: #ffffff;
            text-align: left;
        }

        .document-body table.data-table td,
        .document-body table.data-table th {
            border: 0.6pt solid #444;
            padding: 5pt 8pt;
            vertical-align: top;
            text-align: left;
            font-size: 10pt;
        }

        .document-body table.data-table th {
            background-color: #f1f5f6;
            font-weight: 700;
        }

        .document-body table.signature-table {
            margin-top: 14pt;
            border: none;
            break-inside: avoid;
        }

        .document-body table.signature-table td,
        .document-body table.signature-table th {
            border: none;
            padding: 4pt 2pt;
            vertical-align: top;
            font-size: 10pt;
            font-weight: normal;
        }

        .document-body table.signature-table td:first-child,
        .document-body table.signature-table td:last-child {
            width: 46%;
        }

        .document-body table.signature-table td.sig-spacer,
        .document-body table.signature-table td:nth-child(2) {
            width: 8%;
        }

        .document-body table.signature-table tr:first-child td:first-child,
        .document-body table.signature-table tr:first-child td:last-child {
            padding-bottom: 44pt;
        }

        .document-body table.signature-table tr:last-child td:first-child,
        .document-body table.signature-table tr:last-child td:last-child {
            border-top: 0.75pt solid #333;
            padding-top: 6pt;
        }

        .document-body sup.footnote-ref {
            font-size: 0.74em;
            line-height: 0;
            vertical-align: super;
            margin-left: 1pt;
        }

        .document-body sup.footnote-ref a {
            color: #006666;
            text-decoration: none;
            font-weight: 600;
        }

        mark.preview-merged {
            background-color: #d7f3ef;
            color: #0b4f4a;
            padding: 0 2pt;
            border-radius: 2pt;
        }

        mark.preview-missing {
            background-color: #fef3c7;
            color: #92400e;
            padding: 0 3pt;
            border-radius: 2pt;
            font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
            font-size: 0.88em;
        }

        @media screen and (max-width: 225mm) {

            .legal-document,
            .document-sheet {
                width: auto;
                max-width: 210mm;
                margin: 8mm;
                padding: 16mm;
            }
        }

        @media print {

            html,
            body {
                background: #ffffff;
            }

            .legal-document,
            .document-sheet {
                width: auto;
                min-height: 0;
                margin: 0;
                padding: 0;
                box-shadow: none;
                display: block;
            }

            .doc-running-header,
            .doc-running-footer {
                display: none;
            }

            mark.preview-merged {
                background: transparent;
                color: inherit;
                padding: 0;
            }
        }
    </style>
</head>

<body>
    <article class="legal-document">
        <header class="doc-running-header">
            <span class="header-left">DOME CSP Account administrator appointment Form</span>
            <span class="header-right">version March 2026</span>
        </header>
        <main class="document-body">
            <header class="doc-header">
                <h1 class="doc-title">DOME Marketplace T&amp;Cs acceptance and Cloud Service Provider’s Account
                    Administrator Appointment Form</h1>
                <aside class="callout-box" role="note" aria-label="Instructions">
                    <p>Before designating the Account Administrator, please, read carefully all the document. It gives a
                        detailed overview of the implications of designating an Account Administrator and the acts an
                        Account Administrator can perform on behalf of your entity.</p>
                    <p>Should you have any questions regarding the process or any other topic concerning the Account
                        Administrator and the verifiable credentials generated by the Marketplace, please contact us at
                        <a href="mailto:legal.helpdesk@dome-project.eu">legal.helpdesk@dome-markeplace.org</a>.
                    </p>
                    <footer class="signature-section">
                        <p>You may also wish to read the Practical considerations for the designations of an Account
                            Administrator where you can find some useful additional information.</p>
                    </footer>
                    <p><strong>NOTE:</strong> you must only modify the fields where the input of specific information is
                        needed. Should you modify any other section, sentence or portion of the document, your
                        onboarding request may not be approved.</p>
                </aside>
            </header>
            <section class="doc-section" aria-label="Appointment Form">
                <h2 class="section-heading">ACCEPTANCE OF DOME MARKETPLACE TERMS &amp; CONDITIONS, AND ACCOUNT
                    ADMINISTRATOR APPOINTMENT FORM</h2>
                <table class="form-table date-table">
                    <tbody>
                        <tr>
                            <td class="field-code"><div id="pdf.date">0.1</div></td>
                            <th scope="row">Date</th>
                            <td class="field-value"><time>{{date}}</time></td>
                        </tr>
                    </tbody>
                </table>
                <p><strong>Subject</strong>: Acceptance of DOME Marketplace T&amp;Cs and designation of the Account
                    Administrator in DOME Marketplace</p>
                <p>I, the undersigned,</p>
                <table class="form-table">
                    <tbody>
                        <tr>
                            <td class="field-code"><div id="pdf.legal_rep.name">1.1</div></td>
                            <th scope="row">Name</th>
                            <td class="field-value">{{legal_rep.name}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.legal_rep.surname">1.2</div></td>
                            <th scope="row">Surname</th>
                            <td class="field-value">{{legal_rep.surname}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.legal_rep.id_number">1.3</div></td>
                            <th scope="row">ID card number (optional)</th>
                            <td class="field-value">{{legal_rep.id_number}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.legal_rep.email">1.4</div></td>
                            <th scope="row">email</th>
                            <td class="field-value">{{legal_rep.email}}</td>
                        </tr>
                    </tbody>
                </table>
                <p>acting, as legal representative, for and on behalf of:</p>
                <table class="form-table">
                    <tbody>
                        <tr>
                            <td class="field-code"><div id="pdf.company.legal_name">2.1</div></td>
                            <th scope="row">Entity full legal name<sup class="footnote-ref" data-footnote-id="fn-1"><a
                                        href="#fn-1">1</a></sup></th>
                            <td class="field-value">{{company.legal_name}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.company.registered_address">2.2</div></td>
                            <th scope="row">Registered office full address</th>
                            <td class="field-value">{{company.registered_address}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.company.country_code">2.3</div></td>
                            <th scope="row">Country of incorporation<sup class="footnote-ref" data-footnote-id="fn-2"><a
                                        href="#fn-2">2</a></sup></th>
                            <td class="field-value">{{company.country_code}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.company.tax_id">2.4</div></td>
                            <th scope="row">Tax ID number</th>
                            <td class="field-value">{{company.tax_id}}</td>
                        </tr>
                    </tbody>
                </table>
                <p>(hereinafter, the Company)</p>
                <p>Do hereby:</p>
                <ol>
                    <li style="text-align: justify;">Confirm that I am the legal representative of Company and that I
                        have sufficient powers to sign this form on behalf of Company;</li>
                    <li style="text-align: justify;">After having read and understood the <a
                            href="https://onboard.dome.mycredential.eu/api/files/pbc_365946868/ki9zjz96v7re63n/20250303_dome_t_c_for_cloud_customers_6zqyb3yz7d.pdf"
                            target="_blank" rel="noopener noreferrer">DOME Marketplace Terms and Conditions of
                            contract</a>, confirm that Company accepts them;</li>
                    <li style="text-align: justify;">After having read and understood the information below concerning
                        the designation and the powers of an Account Administrator within the DOME Marketplace, do
                        hereby designate as the DOME Account Administrator of Company:</li>
                </ol>
                <table class="form-table">
                    <tbody>
                        <tr>
                            <td class="field-code"><div id="pdf.account_admin.name">3.1</div></td>
                            <th scope="row">Name</th>
                            <td class="field-value">{{account_admin.name}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.account_admin.surname">3.2</div></td>
                            <th scope="row">Surname</th>
                            <td class="field-value">{{account_admin.surname}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.account_admin.id_number">3.3</div></td>
                            <th scope="row">ID card number (optional)</th>
                            <td class="field-value">{{account_admin.id_number}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.account_admin.postal_address">3.4</div></td>
                            <th scope="row">Complete postal professional address</th>
                            <td class="field-value">{{account_admin.postal_address}}</td>
                        </tr>
                        <tr>
                            <td class="field-code"><div id="pdf.account_admin.email">3.5</div></td>
                            <th scope="row">Email:</th>
                            <td class="field-value">{{account_admin.email}}</td>
                        </tr>
                    </tbody>
                </table>
            </section>
            <section class="doc-section" aria-label="Role and Powers of the Account Administrator">
                <h2 class="section-heading-plain">ROLE AND POWERS OF THE ACCOUNT ADMINISTRATOR</h2>
                <p>The Account Administrator of the Company is the person that any entity willing to onboard on the DOME
                    Marketplace must designate to have a completely operational account in the Marketplace, i.e., an
                    account that enables the publication of the Offerings of the Company.</p>
                <p>Before choosing the individual that you want to designate as Account Administrator, please <u>bear in
                        mind that this role requires a daily involvement in Company's operations in the DOME
                        Marketplace</u>.</p>
                <p>The Account Administrator will be responsible for providing and checking that all the information
                    concerning the entity is accurate and remains up to date at any time.</p>
                <p>The Account Administrator will represent the Company on every aspect of the activity of Company only
                    in the context of Company operations within the DOME Marketplace. That is, every action performed by
                    the Account Administrator will be legally attributed to and binding on the Company.</p>
                <p>The Account Administrator will be entitled to delegate some of its faculties to other individuals,
                    except the capacity of delegation itself: i.e., delegated persons cannot further delegate the
                    representation powers to other. Those delegated persons will also represent the Company always and
                    only within the DOME Marketplace within the limited scope of the delegated powers. act within the
                    Marketplace on behalf of your Company within the scope of the delegation.</p>
                <p>The appointment of the Account Administrator is indefinite in time. Nonetheless, the Company can
                    change its Account Administrator at any moment.</p>
                <p>A single Company can appoint more than one Account Administrator at the same time. For this the
                    Company must fill in and send an appointment form (i.e., this document) for each Account
                    Administrator.</p>
                <p>It is the responsibility of the Company to revoke the designation of an individual as Account
                    Administrator or any other delegated individual within the scope of operation of the DOME
                    Marketplace once the Company does not wish to be represented within the Marketplace by such
                    individuals.</p>
            </section>
            <footer class="signature-section">
                <p>In witness whereof I sign this appointment letter on the date set at the beginning of the letter.</p>
                <table class="signature-table">
                    <tbody>
                        <tr>
                            <td>For and on behalf of Company</td>
                            <td class="sig-spacer"><br></td>
                            <td>Acceptance of the appointment by the Account Administrator</td>
                        </tr>
                        <tr>
                            <td>Signed: Mr./Mrs. {{legal_rep.name}} {{legal_rep.surname}}</td>
                            <td class="sig-spacer"><br></td>
                            <td><strong>Accepted</strong><br>Signed: Mr./Mrs. {{account_admin.name}}
                                {{account_admin.surname}}</td>
                        </tr>
                    </tbody>
                </table>
            </footer>
        </main>
        <section class="doc-footnotes" aria-label="Footnotes">
            <ol class="footnotes-list">
                <li id="fn-1" data-footnote-id="fn-1">Make sure that you include the acronym of the legal form of your
                    company (SA, SAS, SL, LLM GmbH, VB, etc.)</li>
                <li id="fn-2" data-footnote-id="fn-2">For designating the country of incorporation, please use the
                    two-letter country code. You can find such code <a
                        href="https://ec.europa.eu/eurostat/statistics-explained/index.php?title=Glossary:Country_codes"
                        target="_blank" rel="noopener noreferrer">here</a>.</li>
            </ol>
        </section>
        <footer class="doc-running-footer">
            <span class="footer-left"></span>
            <span class="footer-right">Page 1 out of 1</span>
        </footer>
    </article>
</body>

</html>
`

type appointmentData struct {
	Date         string           `json:"date"`
	LegalRep     legalRepData     `json:"legal_rep"`
	CompanyData  companyData      `json:"company"`
	AccountAdmin accountAdminData `json:"account_admin"`
}

type legalRepData struct {
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	IDNumber string `json:"id_number"`
	Email    string `json:"email"`
}

type companyData struct {
	LegalName         string `json:"legal_name"`
	RegisteredAddress string `json:"registered_address"`
	CountryCode       string `json:"country_code"`
	TaxID             string `json:"tax_id"`
}

type accountAdminData struct {
	Name          string `json:"name"`
	Surname       string `json:"surname"`
	IDNumber      string `json:"id_number"`
	PostalAddress string `json:"postal_address"`
	Email         string `json:"email"`
}

func (s *Server) generateAppointmentLetterPDF(current *db.RegistrationRecord) ([]byte, error) {

	now := time.Now()

	appointment := appointmentData{
		Date: now.Format("2006-01-02"),
		LegalRep: legalRepData{
			Name:     current.LRFirstName,
			Surname:  current.LRLastName,
			IDNumber: current.LRIdCard,
			Email:    current.LREmail,
		},
		CompanyData: companyData{
			LegalName:         current.CompanyName,
			RegisteredAddress: current.StreetAddress + ", " + current.PostalCode + " - " + current.City + ", " + current.Country,
			CountryCode:       current.Country,
			TaxID:             current.VatID,
		},
		AccountAdmin: accountAdminData{
			Name:          current.LEARFirstName,
			Surname:       current.LEARLastName,
			IDNumber:      current.LEARIdCard,
			PostalAddress: current.LEARAddress,
			Email:         current.LEAREmail,
		},
	}

	inputJSON, err := json.Marshal(appointment)
	if err != nil {
		return nil, err
	}

	// 1. Render HTML from template + JSON
	renderedHTML, err := pdf.RenderHTML(templateHTML, inputJSON)
	if err != nil {
		return nil, err
	}

	// 2. Generate PDF/A-3B with embedded pdf-data.json + optional HMAC in XMP
	pdfBytes, err := pdf.GeneratePDF(renderedHTML, inputJSON, nil)
	if err != nil {
		return nil, err
	}

	return pdfBytes, nil
}
