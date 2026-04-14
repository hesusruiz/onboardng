package common

type Country struct {
	Code string
	Name string
}

var WorldCountries = []Country{
	{Code: "US", Name: "United States"},
	{Code: "GB", Name: "United Kingdom"},
	{Code: "CA", Name: "Canada"},
	{Code: "AU", Name: "Australia"},
	{Code: "DE", Name: "Germany"},
	{Code: "FR", Name: "France"},
	{Code: "ES", Name: "Spain"},
	{Code: "IT", Name: "Italy"},
	{Code: "NL", Name: "Netherlands"},
	{Code: "BE", Name: "Belgium"},
	{Code: "CH", Name: "Switzerland"},
	{Code: "AT", Name: "Austria"},
	{Code: "SE", Name: "Sweden"},
	{Code: "NO", Name: "Norway"},
	{Code: "DK", Name: "Denmark"},
	{Code: "FI", Name: "Finland"},
	{Code: "IE", Name: "Ireland"},
	{Code: "PT", Name: "Portugal"},
	{Code: "GR", Name: "Greece"},
	{Code: "LU", Name: "Luxembourg"},
	{Code: "JP", Name: "Japan"},
	{Code: "CN", Name: "China"},
	{Code: "IN", Name: "India"},
	{Code: "BR", Name: "Brazil"},
	{Code: "MX", Name: "Mexico"},
	{Code: "ZA", Name: "South Africa"},
	{Code: "AE", Name: "United Arab Emirates"},
	{Code: "SG", Name: "Singapore"},
	{Code: "KR", Name: "South Korea"},
	{Code: "NZ", Name: "New Zealand"},
}

// <option value="AT">Austria</option>
// <option value="BE">Belgium</option>
// <option value="BG">Bulgaria</option>
// <option value="HR">Croatia</option>
// <option value="CY">Cyprus</option>
// <option value="CZ">Czech Republic</option>
// <option value="DK">Denmark</option>
// <option value="EE">Estonia</option>
// <option value="FI">Finland</option>
// <option value="FR">France</option>
// <option value="DE">Germany</option>
// <option value="EL">Greece</option>
// <option value="HU">Hungary</option>
// <option value="IS">Iceland</option>
// <option value="IE">Ireland</option>
// <option value="IT">Italy</option>
// <option value="LV">Latvia</option>
// <option value="LI">Liechtenstein</option>
// <option value="LT">Lithuania</option>
// <option value="LU">Luxembourg</option>
// <option value="MT">Malta</option>
// <option value="NL">Netherlands</option>
// <option value="NO">Norway</option>
// <option value="PL">Poland</option>
// <option value="PT">Portugal</option>
// <option value="RO">Romania</option>
// <option value="SK">Slovakia</option>
// <option value="SI">Slovenia</option>
// <option value="ES">Spain</option>
// <option value="SE">Sweden</option>
var EEACountries = []Country{
	{Code: "AT", Name: "Austria"},
	{Code: "BE", Name: "Belgium"},
	{Code: "BG", Name: "Bulgaria"},
	{Code: "HR", Name: "Croatia"},
	{Code: "CY", Name: "Cyprus"},
	{Code: "CZ", Name: "Czech Republic"},
	{Code: "DK", Name: "Denmark"},
	{Code: "EE", Name: "Estonia"},
	{Code: "FI", Name: "Finland"},
	{Code: "FR", Name: "France"},
	{Code: "DE", Name: "Germany"},
	{Code: "EL", Name: "Greece"},
	{Code: "HU", Name: "Hungary"},
	{Code: "IS", Name: "Iceland"},
	{Code: "IE", Name: "Ireland"},
	{Code: "IT", Name: "Italy"},
	{Code: "LV", Name: "Latvia"},
	{Code: "LI", Name: "Liechtenstein"},
	{Code: "LT", Name: "Lithuania"},
	{Code: "LU", Name: "Luxembourg"},
	{Code: "MT", Name: "Malta"},
	{Code: "NL", Name: "Netherlands"},
	{Code: "NO", Name: "Norway"},
	{Code: "PL", Name: "Poland"},
	{Code: "PT", Name: "Portugal"},
	{Code: "RO", Name: "Romania"},
	{Code: "SK", Name: "Slovakia"},
	{Code: "SI", Name: "Slovenia"},
	{Code: "ES", Name: "Spain"},
	{Code: "SE", Name: "Sweden"},
}

func GetCountryName(code string) string {
	for _, c := range EEACountries {
		if c.Code == code {
			return c.Name
		}
	}
	return ""
}

func IsValidCountry(code string) bool {
	for _, c := range EEACountries {
		if c.Code == code {
			return true
		}
	}
	return false
}
