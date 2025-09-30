package codigos

import "strings"

// Country represents a country
type Country struct {
	Code  string
	Value string
}

// Country codes - ISO 3166-1 alpha-2
const (
	CountryAfghanistan        = "AF"
	CountryAland              = "AX"
	CountryAlbania            = "AL"
	CountryGermany            = "DE"
	CountryAndorra            = "AD"
	CountryAngola             = "AO"
	CountryAnguilla           = "AI"
	CountryAntarctica         = "AQ"
	CountryAntiguaBarbuda     = "AG"
	CountryAruba              = "AW"
	CountrySaudiArabia        = "SA"
	CountryAlgeria            = "DZ"
	CountryArgentina          = "AR"
	CountryArmenia            = "AM"
	CountryAustralia          = "AU"
	CountryAustria            = "AT"
	CountryAzerbaijan         = "AZ"
	CountryBahamas            = "BS"
	CountryBahrain            = "BH"
	CountryBangladesh         = "BD"
	CountryBarbados           = "BB"
	CountryBelgium            = "BE"
	CountryBelize             = "BZ"
	CountryBenin              = "BJ"
	CountryBermuda            = "BM"
	CountryBelarus            = "BY"
	CountryBolivia            = "BO"
	CountryBonaire            = "BQ"
	CountryBosniaHerzegovina  = "BA"
	CountryBotswana           = "BW"
	CountryBrazil             = "BR"
	CountryBrunei             = "BN"
	CountryBulgaria           = "BG"
	CountryBurkinaFaso        = "BF"
	CountryBurundi            = "BI"
	CountryBhutan             = "BT"
	CountryCaboVerde          = "CV"
	CountryCaymanIslands      = "KY"
	CountryCambodia           = "KH"
	CountryCameroon           = "CM"
	CountryCanada             = "CA"
	CountryCentralAfrican     = "CF"
	CountryChad               = "TD"
	CountryChile              = "CL"
	CountryChina              = "CN"
	CountryCyprus             = "CY"
	CountryVatican            = "VA"
	CountryColombia           = "CO"
	CountryComoros            = "KM"
	CountryCongo              = "CG"
	CountryCostaMarfil        = "CI"
	CountryCostaRica          = "CR"
	CountryCroatia            = "HR"
	CountryCuba               = "CU"
	CountryCuracao            = "CW"
	CountryDenmark            = "DK"
	CountryDominica           = "DM"
	CountryDjibouti           = "DJ"
	CountryEcuador            = "EC"
	CountryEgypt              = "EG"
	CountryElSalvador         = "SV"
	CountryUAE                = "AE"
	CountryEritrea            = "ER"
	CountrySlovakia           = "SK"
	CountrySlovenia           = "SI"
	CountrySpain              = "ES"
	CountryUSA                = "US"
	CountryEstonia            = "EE"
	CountryEthiopia           = "ET"
	CountryFiji               = "FJ"
	CountryPhilippines        = "PH"
	CountryFinland            = "FI"
	CountryFrance             = "FR"
	CountryGabon              = "GA"
	CountryGambia             = "GM"
	CountryGeorgia            = "GE"
	CountryGhana              = "GH"
	CountryGibraltar          = "GI"
	CountryGrenada            = "GD"
	CountryGreece             = "GR"
	CountryGreenland          = "GL"
	CountryGuadeloupe         = "GP"
	CountryGuam               = "GU"
	CountryGuatemala          = "GT"
	CountryFrenchGuiana       = "GF"
	CountryGuernsey           = "GG"
	CountryGuinea             = "GN"
	CountryEquatorialGuinea   = "GQ"
	CountryGuineaBissau       = "GW"
	CountryGuyana             = "GY"
	CountryHaiti              = "HT"
	CountryHonduras           = "HN"
	CountryHongKong           = "HK"
	CountryHungary            = "HU"
	CountryIndia              = "IN"
	CountryIndonesia          = "ID"
	CountryIraq               = "IQ"
	CountryIreland            = "IE"
	CountryBouvetIsland       = "BV"
	CountryManIsland          = "IM"
	CountryNorfolkIsland      = "NF"
	CountryIceland            = "IS"
	CountryChristmasIsland    = "CX"
	CountryCocosIslands       = "CC"
	CountryCookIslands        = "CK"
	CountryFaroeIslands       = "FO"
	CountrySouthGeorgia       = "GS"
	CountryHeardMcDonald      = "HM"
	CountryFalklandIslands    = "FK"
	CountryMarianaIslands     = "MP"
	CountryMarshallIslands    = "MH"
	CountryPitcairn           = "PN"
	CountryTurksCaicos        = "TC"
	CountryUSMinorIslands     = "UM"
	CountryVirginIslands      = "VI"
	CountryIsrael             = "IL"
	CountryItaly              = "IT"
	CountryJamaica            = "JM"
	CountryJapan              = "JP"
	CountryJersey             = "JE"
	CountryJordan             = "JO"
	CountryKazakhstan         = "KZ"
	CountryKenya              = "KE"
	CountryKyrgyzstan         = "KG"
	CountryKiribati           = "KI"
	CountryKuwait             = "KW"
	CountryLaos               = "LA"
	CountryLesotho            = "LS"
	CountryLatvia             = "LV"
	CountryLebanon            = "LB"
	CountryLiberia            = "LR"
	CountryLibya              = "LY"
	CountryLiechtenstein      = "LI"
	CountryLithuania          = "LT"
	CountryLuxembourg         = "LU"
	CountryMacao              = "MO"
	CountryMacedonia          = "MK"
	CountryMadagascar         = "MG"
	CountryMalaysia           = "MY"
	CountryMalawi             = "MW"
	CountryMaldives           = "MV"
	CountryMali               = "ML"
	CountryMalta              = "MT"
	CountryMorocco            = "MA"
	CountryMartinique         = "MQ"
	CountryMauritius          = "MU"
	CountryMauritania         = "MR"
	CountryMayotte            = "YT"
	CountryMexico             = "MX"
	CountryMicronesia         = "FM"
	CountryMoldova            = "MD"
	CountryMonaco             = "MC"
	CountryMongolia           = "MN"
	CountryMontenegro         = "ME"
	CountryMontserrat         = "MS"
	CountryMozambique         = "MZ"
	CountryMyanmar            = "MM"
	CountryNamibia            = "NA"
	CountryNauru              = "NR"
	CountryNepal              = "NP"
	CountryNicaragua          = "NI"
	CountryNiger              = "NE"
	CountryNigeria            = "NG"
	CountryNiue               = "NU"
	CountryNorway             = "NO"
	CountryNewCaledonia       = "NC"
	CountryNewZealand         = "NZ"
	CountryOman               = "OM"
	CountryNetherlands        = "NL"
	CountryPakistan           = "PK"
	CountryPalau              = "PW"
	CountryPalestine          = "PS"
	CountryPanama             = "PA"
	CountryPapuaNewGuinea     = "PG"
	CountryParaguay           = "PY"
	CountryPeru               = "PE"
	CountryFrenchPolynesia    = "PF"
	CountryPoland             = "PL"
	CountryPortugal           = "PT"
	CountryPuertoRico         = "PR"
	CountryQatar              = "QA"
	CountryUK                 = "GB"
	CountryNorthKorea         = "KP"
	CountryCzechRepublic      = "CZ"
	CountryKorea              = "KR"
	CountryDRCongo            = "CD"
	CountryDominicanRepublic  = "DO"
	CountryIran               = "IR"
	CountryReunion            = "RE"
	CountryRwanda             = "RW"
	CountryRomania            = "RO"
	CountryRussia             = "RU"
	CountryWesternSahara      = "EH"
	CountrySaintBarthelemy    = "BL"
	CountrySaintMartin        = "MF"
	CountrySolomonIslands     = "SB"
	CountrySamoa              = "WS"
	CountryAmericanSamoa      = "AS"
	CountrySaintKitts         = "KN"
	CountrySanMarino          = "SM"
	CountrySaintPierre        = "PM"
	CountrySaintVincent       = "VC"
	CountrySaintHelena        = "SH"
	CountrySaintLucia         = "LC"
	CountrySaoTome            = "ST"
	CountrySenegal            = "SN"
	CountrySerbia             = "RS"
	CountrySeychelles         = "SC"
	CountrySierraLeone        = "SL"
	CountrySingapore          = "SG"
	CountrySintMaarten        = "SX"
	CountrySyria              = "SY"
	CountrySomalia            = "SO"
	CountrySouthSudan         = "SS"
	CountrySriLanka           = "LK"
	CountrySouthAfrica        = "ZA"
	CountrySudan              = "SD"
	CountrySweden             = "SE"
	CountrySwitzerland        = "CH"
	CountrySuriname           = "SR"
	CountrySvalbard           = "SJ"
	CountrySwaziland          = "SZ"
	CountryThailand           = "TH"
	CountryTaiwan             = "TW"
	CountryTanzania           = "TZ"
	CountryTajikistan         = "TJ"
	CountryBritishIndianOcean = "IO"
	CountryFrenchSouthern     = "TF"
	CountryTimorLeste         = "TL"
	CountryTogo               = "TG"
	CountryTokelau            = "TK"
	CountryTonga              = "TO"
	CountryTrinidadTobago     = "TT"
	CountryTunisia            = "TN"
	CountryTurkmenistan       = "TM"
	CountryTurkey             = "TR"
	CountryTuvalu             = "TV"
	CountryUkraine            = "UA"
	CountryUganda             = "UG"
	CountryUruguay            = "UY"
	CountryUzbekistan         = "UZ"
	CountryVanuatu            = "VU"
	CountryVenezuela          = "VE"
	CountryVietnam            = "VN"
	CountryBritishVirgin      = "VG"
	CountryWallisFutuna       = "WF"
	CountryYemen              = "YE"
	CountryZambia             = "ZM"
	CountryZimbabwe           = "ZW"
)

// Countries is a map of all countries
var Countries = map[string]string{
	"AF": "Afganistán",
	"AX": "Aland",
	"AL": "Albania",
	"DE": "Alemania",
	"AD": "Andorra",
	"AO": "Angola",
	"AI": "Anguila",
	"AQ": "Antártica",
	"AG": "Antigua y Barbuda",
	"AW": "Aruba",
	"SA": "Arabia Saudita",
	"DZ": "Argelia",
	"AR": "Argentina",
	"AM": "Armenia",
	"AU": "Australia",
	"AT": "Austria",
	"AZ": "Azerbaiyán",
	"BS": "Bahamas",
	"BH": "Bahrein",
	"BD": "Bangladesh",
	"BB": "Barbados",
	"BE": "Bélgica",
	"BZ": "Belice",
	"BJ": "Benin",
	"BM": "Bermudas",
	"BY": "Bielorrusia",
	"BO": "Bolivia",
	"BQ": "Bonaire, Sint Eustatius and Saba",
	"BA": "Bosnia-Herzegovina",
	"BW": "Botswana",
	"BR": "Brasil",
	"BN": "Brunei",
	"BG": "Bulgaria",
	"BF": "Burkina Faso",
	"BI": "Burundi",
	"BT": "Bután",
	"CV": "Cabo Verde",
	"KY": "Caimán, Islas",
	"KH": "Camboya",
	"CM": "Camerún",
	"CA": "Canadá",
	"CF": "Centroafricana, República",
	"TD": "Chad",
	"CL": "Chile",
	"CN": "China",
	"CY": "Chipre",
	"VA": "Ciudad del Vaticano",
	"CO": "Colombia",
	"KM": "Comoras",
	"CG": "Congo",
	"CI": "Costa de Marfil",
	"CR": "Costa Rica",
	"HR": "Croacia",
	"CU": "Cuba",
	"CW": "Curazao",
	"DK": "Dinamarca",
	"DM": "Dominica",
	"DJ": "Djibouti",
	"EC": "Ecuador",
	"EG": "Egipto",
	"SV": "El Salvador",
	"AE": "Emiratos Arabes Unidos",
	"ER": "Eritrea",
	"SK": "Eslovaquia",
	"SI": "Eslovenia",
	"ES": "España",
	"US": "Estados Unidos",
	"EE": "Estonia",
	"ET": "Etiopia",
	"FJ": "Fiji",
	"PH": "Filipinas",
	"FI": "Finlandia",
	"FR": "Francia",
	"GA": "Gabón",
	"GM": "Gambia",
	"GE": "Georgia",
	"GH": "Ghana",
	"GI": "Gibraltar",
	"GD": "Granada",
	"GR": "Grecia",
	"GL": "Groenlandia",
	"GP": "Guadalupe",
	"GU": "Guam",
	"GT": "Guatemala",
	"GF": "Guayana Francesa",
	"GG": "Guernsey",
	"GN": "Guinea",
	"GQ": "Guinea Ecuatorial",
	"GW": "Guinea-Bissau",
	"GY": "Guyana",
	"HT": "Haiti",
	"HN": "Honduras",
	"HK": "Hong Kong",
	"HU": "Hungría",
	"IN": "India",
	"ID": "Indonesia",
	"IQ": "Irak",
	"IE": "Irlanda",
	"BV": "Isla Bouvet",
	"IM": "Isla de Man",
	"NF": "Isla Norfolk",
	"IS": "Islandia",
	"CX": "Islas Navidad",
	"CC": "Islas Cocos",
	"CK": "Islas Cook",
	"FO": "Islas Faroe",
	"GS": "Islas Georgias d. S.-Sandwich d. S.",
	"HM": "Islas Heard y McDonald",
	"FK": "Islas Malvinas (Falkland)",
	"MP": "Islas Marianas del Norte",
	"MH": "Islas Marshall",
	"PN": "Islas Pitcairn",
	"TC": "Islas Turcas y Caicos",
	"UM": "Islas Ultramarinas de E.E.U.U",
	"VI": "Islas Vírgenes",
	"IL": "Israel",
	"IT": "Italia",
	"JM": "Jamaica",
	"JP": "Japón",
	"JE": "Jersey",
	"JO": "Jordania",
	"KZ": "Kazajistán",
	"KE": "Kenia",
	"KG": "Kirguistán",
	"KI": "Kiribati",
	"KW": "Kuwait",
	"LA": "Laos, República Democrática",
	"LS": "Lesotho",
	"LV": "Letonia",
	"LB": "Líbano",
	"LR": "Liberia",
	"LY": "Libia",
	"LI": "Liechtenstein",
	"LT": "Lituania",
	"LU": "Luxemburgo",
	"MO": "Macao",
	"MK": "Macedonia",
	"MG": "Madagascar",
	"MY": "Malasia",
	"MW": "Malawi",
	"MV": "Maldivas",
	"ML": "Malí",
	"MT": "Malta",
	"MA": "Marruecos",
	"MQ": "Martinica e.a.",
	"MU": "Mauricio",
	"MR": "Mauritania",
	"YT": "Mayotte",
	"MX": "México",
	"FM": "Micronesia",
	"MD": "Moldavia, República de",
	"MC": "Mónaco",
	"MN": "Mongolia",
	"ME": "Montenegro",
	"MS": "Montserrat",
	"MZ": "Mozambique",
	"MM": "Myanmar",
	"NA": "Namibia",
	"NR": "Nauru",
	"NP": "Nepal",
	"NI": "Nicaragua",
	"NE": "Níger",
	"NG": "Nigeria",
	"NU": "Niue",
	"NO": "Noruega",
	"NC": "Nueva Caledonia",
	"NZ": "Nueva Zelanda",
	"OM": "Omán",
	"NL": "Países Bajos",
	"PK": "Pakistán",
	"PW": "Palaos",
	"PS": "Palestina",
	"PA": "Panamá",
	"PG": "Papúa, Nueva Guinea",
	"PY": "Paraguay",
	"PE": "Perú",
	"PF": "Polinesia Francesa",
	"PL": "Polonia",
	"PT": "Portugal",
	"PR": "Puerto Rico",
	"QA": "Qatar",
	"GB": "Reino Unido",
	"KP": "Rep. Democrática popular de Corea",
	"CZ": "República Checa",
	"KR": "República de Corea",
	"CD": "República Democrática del Congo",
	"DO": "República Dominicana",
	"IR": "República Islámica de Irán",
	"RE": "Reunión",
	"RW": "Ruanda",
	"RO": "Rumania",
	"RU": "Rusia",
	"EH": "Sahara Occidental",
	"BL": "Saint Barthélemy",
	"MF": "Saint Martin (French part)",
	"SB": "Salomón, Islas",
	"WS": "Samoa",
	"AS": "Samoa Americana",
	"KN": "San Cristóbal y Nieves",
	"SM": "San Marino",
	"PM": "San Pedro y Miquelón",
	"VC": "San Vicente y las Granadinas",
	"SH": "Santa Elena",
	"LC": "Santa Lucia",
	"ST": "Santo Tomé y Príncipe",
	"SN": "Senegal",
	"RS": "Serbia",
	"SC": "Seychelles",
	"SL": "Sierra Leona",
	"SG": "Singapur",
	"SX": "Sint Maarten (Dutch part)",
	"SY": "Siria",
	"SO": "Somalia",
	"SS": "South Sudan",
	"LK": "Sri Lanka",
	"ZA": "Sudáfrica",
	"SD": "Sudán",
	"SE": "Suecia",
	"CH": "Suiza",
	"SR": "Surinám",
	"SJ": "Svalbard y Jan Mayen",
	"SZ": "Swazilandia",
	"TH": "Tailandia",
	"TW": "Taiwan, Provincia de China",
	"TZ": "Tanzania, República Unida de",
	"TJ": "Tayikistán",
	"IO": "Territorio Británico Océano Indico",
	"TF": "Territorios Australes Franceses",
	"TL": "Timor Oriental",
	"TG": "Togo",
	"TK": "Tokelau",
	"TO": "Tonga",
	"TT": "Trinidad y Tobago",
	"TN": "Túnez",
	"TM": "Turkmenistán",
	"TR": "Turquía",
	"TV": "Tuvalu",
	"UA": "Ucrania",
	"UG": "Uganda",
	"UY": "Uruguay",
	"UZ": "Uzbekistán",
	"VU": "Vanuatu",
	"VE": "Venezuela",
	"VN": "Vietnam",
	"VG": "Islas Vírgenes Británicas",
	"WF": "Wallis y Fortuna, Islas",
	"YE": "Yemen",
	"ZM": "Zambia",
	"ZW": "Zimbabue",
}

// GetCountryName returns the name of a country by code
func GetCountryName(code string) (string, bool) {
	name, exists := Countries[code]
	return name, exists
}

// GetCountryCode returns the code for a country by name (case-insensitive)
func GetCountryCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range Countries {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllCountries returns a slice of all countries
func GetAllCountries() []Country {
	countries := make([]Country, 0, len(Countries))
	for code, value := range Countries {
		countries = append(countries, Country{
			Code:  code,
			Value: value,
		})
	}
	return countries
}

// IsValidCountry checks if a country code is valid
func IsValidCountry(code string) bool {
	_, exists := Countries[code]
	return exists
}
