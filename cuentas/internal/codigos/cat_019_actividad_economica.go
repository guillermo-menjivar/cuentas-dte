package codigos

import "strings"

// EconomicActivity represents an economic activity code
type EconomicActivity struct {
	Code  string
	Value string
}

// Economic activity codes - Agriculture, Livestock, Forestry and Fishing
const (
	// AGRICULTURA, GANADERÍA, SILVICULTURA Y PESCA
	// PRODUCCIÓN AGRÍCOLA, PECUARIA, CAZA Y ACTIVIDADES DE SERVICIOS CONEXAS
	ActEcon01111 = "01111"
	ActEcon01112 = "01112"
	ActEcon01113 = "01113"
	ActEcon01114 = "01114"
	ActEcon01119 = "01119"
	ActEcon01120 = "01120"
	ActEcon01131 = "01131"
	ActEcon01132 = "01132"
	ActEcon01133 = "01133"
	ActEcon01134 = "01134"
	ActEcon01140 = "01140"
	ActEcon01150 = "01150"
	ActEcon01161 = "01161"
	ActEcon01162 = "01162"
	ActEcon01191 = "01191"
	ActEcon01192 = "01192"
	ActEcon01199 = "01199"
	ActEcon01220 = "01220"
	ActEcon01230 = "01230"
	ActEcon01240 = "01240"
	ActEcon01251 = "01251"
	ActEcon01252 = "01252"
	ActEcon01260 = "01260"
	ActEcon01271 = "01271"
	ActEcon01272 = "01272"
	ActEcon01281 = "01281"
	ActEcon01282 = "01282"
	ActEcon01291 = "01291"
	ActEcon01292 = "01292"
	ActEcon01299 = "01299"
	ActEcon01300 = "01300"
	ActEcon01301 = "01301"

	// Food processing continued
	ActEcon10613 = "10613"
	ActEcon10621 = "10621"
	ActEcon10628 = "10628"
	ActEcon10711 = "10711"
	ActEcon10712 = "10712"
	ActEcon10713 = "10713"
	ActEcon10721 = "10721"
	ActEcon10722 = "10722"
	ActEcon10723 = "10723"
	ActEcon10724 = "10724"
	ActEcon10730 = "10730"
	ActEcon10740 = "10740"
	ActEcon10750 = "10750"
	ActEcon10791 = "10791"
	ActEcon10792 = "10792"
	ActEcon10793 = "10793"
	ActEcon10794 = "10794"
	ActEcon10799 = "10799"
	ActEcon10800 = "10800"

	// ELABORACIÓN DE BEBIDAS
	ActEcon11012 = "11012"
	ActEcon11020 = "11020"
	ActEcon11030 = "11030"
	ActEcon11041 = "11041"
	ActEcon11042 = "11042"
	ActEcon11043 = "11043"
	ActEcon11048 = "11048"
	ActEcon11049 = "11049"

	// ELABORACIÓN DE PRODUCTOS DE TABACO
	ActEcon12000 = "12000"

	// FABRICACIÓN DE PRODUCTOS TEXTILES
	ActEcon13111 = "13111"
	ActEcon13112 = "13112"
	ActEcon13120 = "13120"
	ActEcon13130 = "13130"
	ActEcon13910 = "13910"
	ActEcon13921 = "13921"
	ActEcon13922 = "13922"
	ActEcon13929 = "13929"
	ActEcon13930 = "13930"
	ActEcon13941 = "13941"
	ActEcon13942 = "13942"
	ActEcon13948 = "13948"
	ActEcon13991 = "13991"
	ActEcon13992 = "13992"
	ActEcon13999 = "13999"

	// FABRICACIÓN DE PRENDAS DE VESTIR
	ActEcon14101 = "14101"
	ActEcon14102 = "14102"
	ActEcon14103 = "14103"
	ActEcon14104 = "14104"
	ActEcon14105 = "14105"
	ActEcon14106 = "14106"
	ActEcon14108 = "14108"
	ActEcon14109 = "14109"
	ActEcon14200 = "14200"
	ActEcon14301 = "14301"
	ActEcon14302 = "14302"
	ActEcon14309 = "14309"

	// FABRICACIÓN DE CUEROS Y PRODUCTOS CONEXOS
	ActEcon15110 = "15110"
	ActEcon15121 = "15121"
	ActEcon15122 = "15122"
	ActEcon15123 = "15123"
	ActEcon15128 = "15128"
	ActEcon15201 = "15201"
	ActEcon15202 = "15202"
	ActEcon15208 = "15208"

	// PRODUCCIÓN DE MADERA Y FABRICACIÓN DE PRODUCTOS DE MADERA Y CORCHO
	ActEcon16100 = "16100"
	ActEcon16210 = "16210"
	ActEcon16220 = "16220"
	ActEcon16230 = "16230"

	ActEcon16292 = "16292"
	ActEcon16299 = "16299"

	// FABRICACIÓN DE PAPEL Y DE PRODUCTOS DE PAPEL
	ActEcon17010 = "17010"
	ActEcon17020 = "17020"
	ActEcon17091 = "17091"
	ActEcon17092 = "17092"

	// IMPRESIÓN Y REPRODUCCIÓN DE GRABACIONES
	ActEcon18110 = "18110"
	ActEcon18120 = "18120"
	ActEcon18200 = "18200"

	// FABRICACIÓN DE COQUE Y DE PRODUCTOS DE LA REFINACIÓN DE PETRÓLEO
	ActEcon19100 = "19100"
	ActEcon19201 = "19201"
	ActEcon19202 = "19202"

	// FABRICACIÓN DE SUSTANCIAS Y PRODUCTOS QUÍMICOS
	ActEcon20111 = "20111"
	ActEcon20112 = "20112"
	ActEcon20113 = "20113"
	ActEcon20114 = "20114"
	ActEcon20119 = "20119"
	ActEcon20120 = "20120"
	ActEcon20130 = "20130"
	ActEcon20210 = "20210"
	ActEcon20220 = "20220"
	ActEcon20231 = "20231"
	ActEcon20232 = "20232"
	ActEcon20291 = "20291"
	ActEcon20292 = "20292"
	ActEcon20299 = "20299"
	ActEcon20300 = "20300"

	// FABRICACIÓN DE PRODUCTOS FARMACÉUTICOS
	ActEcon21001 = "21001"
	ActEcon21008 = "21008"

	// FABRICACIÓN DE PRODUCTOS DE CAUCHO Y PLÁSTICO
	ActEcon22110 = "22110"
	ActEcon22190 = "22190"
	ActEcon22201 = "22201"
	ActEcon22202 = "22202"
	ActEcon22208 = "22208"
	ActEcon22209 = "22209"

	// FABRICACIÓN DE PRODUCTOS MINERALES NO METÁLICOS
	ActEcon23101 = "23101"
	ActEcon23102 = "23102"
	ActEcon23108 = "23108"
	ActEcon23109 = "23109"
	ActEcon23910 = "23910"
	ActEcon23920 = "23920"
	ActEcon23931 = "23931"
	ActEcon23932 = "23932"
	ActEcon23940 = "23940"
	ActEcon23950 = "23950"
	ActEcon23960 = "23960"
	ActEcon23990 = "23990"

	// FABRICACIÓN DE METALES COMUNES
	ActEcon24100 = "24100"
	ActEcon24200 = "24200"
	ActEcon24310 = "24310"
	ActEcon24320 = "24320"

	// FABRICACIÓN DE PRODUCTOS DERIVADOS DE METAL
	ActEcon25111 = "25111"
	ActEcon25118 = "25118"
	ActEcon25120 = "25120"
	ActEcon25130 = "25130"
	ActEcon25200 = "25200"
	ActEcon25910 = "25910"
	ActEcon25920 = "25920"
	ActEcon25930 = "25930"
	ActEcon25991 = "25991"

	// Add these constants after ActEcon25991:

	ActEcon25992 = "25992"
	ActEcon25999 = "25999"

	// FABRICACIÓN DE PRODUCTOS DE INFORMÁTICA, ELECTRÓNICA Y ÓPTICA
	ActEcon26100 = "26100"
	ActEcon26200 = "26200"
	ActEcon26300 = "26300"
	ActEcon26400 = "26400"
	ActEcon26510 = "26510"
	ActEcon26520 = "26520"
	ActEcon26600 = "26600"
	ActEcon26700 = "26700"
	ActEcon26800 = "26800"

	// FABRICACIÓN DE EQUIPO ELÉCTRICO
	ActEcon27100 = "27100"
	ActEcon27200 = "27200"
	ActEcon27310 = "27310"
	ActEcon27320 = "27320"
	ActEcon27330 = "27330"
	ActEcon27400 = "27400"
	ActEcon27500 = "27500"
	ActEcon27900 = "27900"

	// FABRICACIÓN DE MAQUINARIA Y EQUIPO NCP
	ActEcon28110 = "28110"
	ActEcon28120 = "28120"
	ActEcon28130 = "28130"
	ActEcon28140 = "28140"
	ActEcon28150 = "28150"
	ActEcon28160 = "28160"
	ActEcon28170 = "28170"
	ActEcon28180 = "28180"
	ActEcon28190 = "28190"
	ActEcon28210 = "28210"
	ActEcon28220 = "28220"
	ActEcon28230 = "28230"
	ActEcon28240 = "28240"
	ActEcon28250 = "28250"
	ActEcon28260 = "28260"
	ActEcon28291 = "28291"
	ActEcon28299 = "28299"

	// FABRICACIÓN DE VEHÍCULOS AUTOMOTORES, REMOLQUES Y SEMIRREMOLQUES
	ActEcon29100 = "29100"
	ActEcon29200 = "29200"
	ActEcon29300 = "29300"

	// FABRICACIÓN DE OTROS TIPOS DE EQUIPO DE TRANSPORTE
	ActEcon30110 = "30110"
	ActEcon30120 = "30120"
	ActEcon30200 = "30200"
	ActEcon30300 = "30300"
	ActEcon30400 = "30400"
	ActEcon30910 = "30910"
	ActEcon30920 = "30920"
	ActEcon30990 = "30990"

	// FABRICACIÓN DE MUEBLES
	ActEcon31001 = "31001"
	ActEcon31002 = "31002"
	ActEcon31008 = "31008"
	ActEcon31009 = "31009"

	// OTRAS INDUSTRIAS MANUFACTURERAS
	ActEcon32110 = "32110"
	ActEcon32120 = "32120"
	ActEcon32200 = "32200"
	ActEcon32301 = "32301"
	ActEcon32308 = "32308"
	ActEcon32401 = "32401"
	ActEcon32402 = "32402"
	ActEcon32409 = "32409"
	ActEcon32500 = "32500"
	ActEcon32901 = "32901"
	ActEcon32902 = "32902"
	ActEcon32903 = "32903"
	ActEcon32904 = "32904"
)

// EconomicActivities is a map of all economic activity codes
var EconomicActivities = map[string]string{
	ActEcon01111: "Cultivo de cereales excepto arroz y para forrajes",
	ActEcon01112: "Cultivo de legumbres",
	ActEcon01113: "Cultivo de semillas oleaginosas",
	ActEcon01114: "Cultivo de plantas para la preparación de semillas",
	ActEcon01119: "Cultivo de otros cereales excepto arroz y forrajeros n.c.p.",
	ActEcon01120: "Cultivo de arroz",
	ActEcon01131: "Cultivo de raíces y tubérculos",
	ActEcon01132: "Cultivo de brotes, bulbos, vegetales tubérculos y cultivos similares",
	ActEcon01133: "Cultivo hortícola de fruto",
	ActEcon01134: "Cultivo de hortalizas de hoja y otras hortalizas ncp",
	ActEcon01140: "Cultivo de caña de azúcar",
	ActEcon01150: "Cultivo de tabaco",
	ActEcon01161: "Cultivo de algodón",
	ActEcon01162: "Cultivo de fibras vegetales excepto algodón",
	ActEcon01191: "Cultivo de plantas no perennes para la producción de semillas y flores",
	ActEcon01192: "Cultivo de cereales y pastos para la alimentación animal",
	ActEcon01199: "Producción de cultivos no estacionales ncp",
	ActEcon01220: "Cultivo de frutas tropicales",
	ActEcon01230: "Cultivo de cítricos",
	ActEcon01240: "Cultivo de frutas de pepita y hueso",
	ActEcon01251: "Cultivo de frutas ncp",
	ActEcon01252: "Cultivo de otros frutos y nueces de árboles y arbustos",
	ActEcon01260: "Cultivo de frutos oleaginosos",
	ActEcon01271: "Cultivo de café",
	ActEcon01272: "Cultivo de plantas para la elaboración de bebidas excepto café",
	ActEcon01281: "Cultivo de especias y aromáticas",
	ActEcon01282: "Cultivo de plantas para la obtención de productos medicinales y farmacéuticos",
	ActEcon01291: "Cultivo de árboles de hule (caucho) para la obtención de látex",
	ActEcon01292: "Cultivo de plantas para la obtención de productos químicos y colorantes",
	ActEcon01299: "Producción de cultivos perennes ncp",
	ActEcon01300: "Propagación de plantas",
	ActEcon01301: "Cultivo de plantas y flores ornamentales",

	ActEcon10613: "Servicios de beneficiado de productos agrícolas ncp (excluye Beneficio de azúcar rama 1072 y beneficio de café rama 0163)",
	ActEcon10621: "Fabricación de almidón",
	ActEcon10628: "Servicio de molienda de maíz húmedo molino para nixtamal",
	ActEcon10711: "Elaboración de tortillas",
	ActEcon10712: "Fabricación de pan, galletas y barquillos",
	ActEcon10713: "Fabricación de repostería",
	ActEcon10721: "Ingenios azucareros",
	ActEcon10722: "Molienda de caña de azúcar para la elaboración de dulces",
	ActEcon10723: "Elaboración de jarabes de azúcar y otros similares",
	ActEcon10724: "Maquilado de azúcar de caña",
	ActEcon10730: "Fabricación de cacao, chocolates y productos de confitería",
	ActEcon10740: "Elaboración de macarrones, fideos, y productos farináceos similares",
	ActEcon10750: "Elaboración de comidas y platos preparados para la reventa en",
	ActEcon10791: "Elaboración de productos de café",
	ActEcon10792: "Elaboración de especias, sazonadores y condimentos",
	ActEcon10793: "Elaboración de sopas, cremas y consomé",
	ActEcon10794: "Fabricación de bocadillos tostados y/o fritos",
	ActEcon10799: "Elaboración de productos alimenticios ncp",
	ActEcon10800: "Elaboración de alimentos preparados para animales",
	ActEcon11012: "Fabricación de aguardiente y licores",
	ActEcon11020: "Elaboración de vinos",
	ActEcon11030: "Fabricación de cerveza",
	ActEcon11041: "Fabricación de aguas gaseosas",
	ActEcon11042: "Fabricación y envasado de agua",
	ActEcon11043: "Elaboración de refrescos",
	ActEcon11048: "Maquilado de aguas gaseosas",
	ActEcon11049: "Elaboración de bebidas no alcohólicas",
	ActEcon12000: "Elaboración de productos de tabaco",
	ActEcon13111: "Preparación de fibras textiles",
	ActEcon13112: "Fabricación de hilados",
	ActEcon13120: "Fabricación de telas",
	ActEcon13130: "Acabado de productos textiles",
	ActEcon13910: "Fabricación de tejidos de punto y ganchillo",
	ActEcon13921: "Fabricación de productos textiles para el hogar",
	ActEcon13922: "Sacos, bolsas y otros artículos textiles",
	ActEcon13929: "Fabricación de artículos confeccionados con materiales textiles, excepto prendas de vestir n.c.p",
	ActEcon13930: "Fabricación de tapices y alfombras",
	ActEcon13941: "Fabricación de cuerdas de henequén y otras fibras naturales (lazos, pitas)",
	ActEcon13942: "Fabricación de redes de diversos materiales",
	ActEcon13948: "Maquilado de productos trenzables de cualquier material (petates, sillas, etc.)",
	ActEcon13991: "Fabricación de adornos, etiquetas y otros artículos para prendas de vestir",
	ActEcon13992: "Servicio de bordados en artículos y prendas de tela",
	ActEcon13999: "Fabricación de productos textiles ncp",
	ActEcon14101: "Fabricación de ropa interior, para dormir y similares",
	ActEcon14102: "Fabricación de ropa para niños",
	ActEcon14103: "Fabricación de prendas de vestir para ambos sexos",
	ActEcon14104: "Confección de prendas a medida",
	ActEcon14105: "Fabricación de prendas de vestir para deportes",
	ActEcon14106: "Elaboración de artesanías de uso personal confeccionadas especialmente de materiales textiles",
	ActEcon14108: "Maquilado de prendas de vestir, accesorios y otros",
	ActEcon14109: "Fabricación de prendas y accesorios de vestir n.c.p.",
	ActEcon14200: "Fabricación de artículos de piel",
	ActEcon14301: "Fabricación de calcetines, calcetas, medias (panty house) y otros similares",
	ActEcon14302: "Fabricación de ropa interior de tejido de punto",
	ActEcon14309: "Fabricación de prendas de vestir de tejido de punto ncp",
	ActEcon15110: "Curtido y adobo de cueros; adobo y teñido de pieles",
	ActEcon15121: "Fabricación de maletas, bolsos de mano y otros artículos de marroquinería",
	ActEcon15122: "Fabricación de monturas, accesorios y vainas talabartería",
	ActEcon15123: "Fabricación de artesanías principalmente de cuero natural y sintético",
	ActEcon15128: "Maquilado de artículos de cuero natural, sintético y de otros materiales",
	ActEcon15201: "Fabricación de calzado",
	ActEcon15202: "Fabricación de partes y accesorios de calzado",
	ActEcon15208: "Maquilado de partes y accesorios de calzado",
	ActEcon16100: "Aserradero y acepilladura de madera",
	ActEcon16210: "Fabricación de madera laminada, terciada, enchapada y contrachapada, paneles para la construcción",
	ActEcon16220: "Fabricación de partes y piezas de carpintería para edificios y construcciones",
	ActEcon16230: "Fabricación de envases y recipientes de madera",

	ActEcon16292: "Fabricación de artesanías de madera, semillas, materiales trenzables",
	ActEcon16299: "Fabricación de productos de madera, corcho, paja y materiales trenzables ncp",
	ActEcon17010: "Fabricación de pasta de madera, papel y cartón",
	ActEcon17020: "Fabricación de papel y cartón ondulado y envases de papel y cartón",
	ActEcon17091: "Fabricación de artículos de papel y cartón de uso personal y doméstico",
	ActEcon17092: "Fabricación de productos de papel ncp",
	ActEcon18110: "Impresión",
	ActEcon18120: "Servicios relacionados con la impresión",
	ActEcon18200: "Reproducción de grabaciones",
	ActEcon19100: "Fabricación de productos de hornos de coque",
	ActEcon19201: "Fabricación de combustible",
	ActEcon19202: "Fabricación de aceites y lubricantes",
	ActEcon20111: "Fabricación de materias primas para la fabricación de colorantes",
	ActEcon20112: "Fabricación de materiales curtientes",
	ActEcon20113: "Fabricación de gases industriales",
	ActEcon20114: "Fabricación de alcohol etílico",
	ActEcon20119: "Fabricación de sustancias químicas básicas",
	ActEcon20120: "Fabricación de abonos y fertilizantes",
	ActEcon20130: "Fabricación de plástico y caucho en formas primarias",
	ActEcon20210: "Fabricación de plaguicidas y otros productos químicos de uso agropecuario",
	ActEcon20220: "Fabricación de pinturas, barnices y productos de revestimiento similares; tintas de imprenta y masillas",
	ActEcon20231: "Fabricación de jabones, detergentes y similares para limpieza",
	ActEcon20232: "Fabricación de perfumes, cosméticos y productos de higiene y cuidado personal, incluyendo tintes, champú, etc.",
	ActEcon20291: "Fabricación de tintas y colores para escribir y pintar; fabricación de cintas para impresoras",
	ActEcon20292: "Fabricación de productos pirotécnicos, explosivos y municiones",
	ActEcon20299: "Fabricación de productos químicos n.c.p.",
	ActEcon20300: "Fabricación de fibras artificiales",
	ActEcon21001: "Manufactura de productos farmacéuticos, sustancias químicas y productos botánicos",
	ActEcon21008: "Maquilado de medicamentos",
	ActEcon22110: "Fabricación de cubiertas y cámaras; renovación y recauchutado de cubiertas",
	ActEcon22190: "Fabricación de otros productos de caucho",
	ActEcon22201: "Fabricación de envases plásticos",
	ActEcon22202: "Fabricación de productos plásticos para uso personal o doméstico",
	ActEcon22208: "Maquila de plásticos",
	ActEcon22209: "Fabricación de productos plásticos n.c.p.",
	ActEcon23101: "Fabricación de vidrio",
	ActEcon23102: "Fabricación de recipientes y envases de vidrio",
	ActEcon23108: "Servicio de maquilado",
	ActEcon23109: "Fabricación de productos de vidrio ncp",
	ActEcon23910: "Fabricación de productos refractarios",
	ActEcon23920: "Fabricación de productos de arcilla para la construcción",
	ActEcon23931: "Fabricación de productos de cerámica y porcelana no refractaria",
	ActEcon23932: "Fabricación de productos de cerámica y porcelana ncp",
	ActEcon23940: "Fabricación de cemento, cal y yeso",
	ActEcon23950: "Fabricación de artículos de hormigón, cemento y yeso",
	ActEcon23960: "Corte, tallado y acabado de la piedra",
	ActEcon23990: "Fabricación de productos minerales no metálicos ncp",
	ActEcon24100: "Industrias básicas de hierro y acero",
	ActEcon24200: "Fabricación de productos primarios de metales preciosos y metales no ferrosos",
	ActEcon24310: "Fundición de hierro y acero",
	ActEcon24320: "Fundición de metales no ferrosos",
	ActEcon25111: "Fabricación de productos metálicos para uso estructural",
	ActEcon25118: "Servicio de maquila para la fabricación de estructuras metálicas",
	ActEcon25120: "Fabricación de tanques, depósitos y recipientes de metal",
	ActEcon25130: "Fabricación de generadores de vapor, excepto calderas de agua caliente para calefacción central",
	ActEcon25200: "Fabricación de armas y municiones",
	ActEcon25910: "Forjado, prensado, estampado y laminado de metales; pulvimetalurgia",
	ActEcon25920: "Tratamiento y revestimiento de metales",
	ActEcon25930: "Fabricación de artículos de cuchillería, herramientas de mano y artículos de ferretería",
	ActEcon25991: "Fabricación de envases y artículos conexos de metal",

	// Add these entries to the EconomicActivities map:

	ActEcon25992: "Fabricación de artículos metálicos de uso personal y/o doméstico",
	ActEcon25999: "Fabricación de productos elaborados de metal ncp",
	ActEcon26100: "Fabricación de componentes electrónicos",
	ActEcon26200: "Fabricación de computadoras y equipo conexo",
	ActEcon26300: "Fabricación de equipo de comunicaciones",
	ActEcon26400: "Fabricación de aparatos electrónicos de consumo para audio, video radio y televisión",
	ActEcon26510: "Fabricación de instrumentos y aparatos para medir, verificar, ensayar, navegar y de control de procesos industriales",
	ActEcon26520: "Fabricación de relojes y piezas de relojes",
	ActEcon26600: "Fabricación de equipo médico de irradiación y equipo electrónico de uso médico y terapéutico",
	ActEcon26700: "Fabricación de instrumentos de óptica y equipo fotográfico",
	ActEcon26800: "Fabricación de medios magnéticos y ópticos",
	ActEcon27100: "Fabricación de motores, generadores, transformadores eléctricos, aparatos de distribución y control de electricidad",
	ActEcon27200: "Fabricación de pilas, baterías y acumuladores",
	ActEcon27310: "Fabricación de cables de fibra óptica",
	ActEcon27320: "Fabricación de otros hilos y cables eléctricos",
	ActEcon27330: "Fabricación de dispositivos de cableados",
	ActEcon27400: "Fabricación de equipo eléctrico de iluminación",
	ActEcon27500: "Fabricación de aparatos de uso doméstico",
	ActEcon27900: "Fabricación de otros tipos de equipo eléctrico",
	ActEcon28110: "Fabricación de motores y turbinas, excepto motores para aeronaves, vehículos automotores y motocicletas",
	ActEcon28120: "Fabricación de equipo hidráulico",
	ActEcon28130: "Fabricación de otras bombas, compresores, grifos y válvulas",
	ActEcon28140: "Fabricación de cojinetes, engranajes, trenes de engranajes y piezas de transmisión",
	ActEcon28150: "Fabricación de hornos y quemadores",
	ActEcon28160: "Fabricación de equipo de elevación y manipulación",
	ActEcon28170: "Fabricación de maquinaria y equipo de oficina",
	ActEcon28180: "Fabricación de herramientas manuales",
	ActEcon28190: "Fabricación de otros tipos de maquinaria de uso general",
	ActEcon28210: "Fabricación de maquinaria agropecuaria y forestal",
	ActEcon28220: "Fabricación de máquinas para conformar metales y maquinaria herramienta",
	ActEcon28230: "Fabricación de maquinaria metalúrgica",
	ActEcon28240: "Fabricación de maquinaria para la explotación de minas y canteras y para obras de construcción",
	ActEcon28250: "Fabricación de maquinaria para la elaboración de alimentos, bebidas y tabaco",
	ActEcon28260: "Fabricación de maquinaria para la elaboración de productos textiles, prendas de vestir y cueros",
	ActEcon28291: "Fabricación de máquinas para imprenta",
	ActEcon28299: "Fabricación de maquinaria de uso especial ncp",
	ActEcon29100: "Fabricación vehículos automotores",
	ActEcon29200: "Fabricación de carrocerías para vehículos automotores; fabricación de remolques y semirremolques",
	ActEcon29300: "Fabricación de partes, piezas y accesorios para vehículos automotores",
	ActEcon30110: "Fabricación de buques",
	ActEcon30120: "Construcción y reparación de embarcaciones de recreo",
	ActEcon30200: "Fabricación de locomotoras y de material rodante",
	ActEcon30300: "Fabricación de aeronaves y naves espaciales",
	ActEcon30400: "Fabricación de vehículos militares de combate",
	ActEcon30910: "Fabricación de motocicletas",
	ActEcon30920: "Fabricación de bicicletas y sillones de ruedas para inválidos",
	ActEcon30990: "Fabricación de equipo de transporte ncp",
	ActEcon31001: "Fabricación de colchones y somier",
	ActEcon31002: "Fabricación de muebles y otros productos de madera a medida",
	ActEcon31008: "Servicios de maquilado de muebles",
	ActEcon31009: "Fabricación de muebles ncp",
	ActEcon32110: "Fabricación de joyas platerías y joyerías",
	ActEcon32120: "Fabricación de joyas de imitación (fantasía) y artículos conexos",
	ActEcon32200: "Fabricación de instrumentos musicales",
	ActEcon32301: "Fabricación de artículos de deporte",
	ActEcon32308: "Servicio de maquila de productos deportivos",
	ActEcon32401: "Fabricación de juegos de mesa y de salón",
	ActEcon32402: "Servicio de maquilado de juguetes y juegos",
	ActEcon32409: "Fabricación de juegos y juguetes n.c.p.",
	ActEcon32500: "Fabricación de instrumentos y materiales médicos y odontológicos",
	ActEcon32901: "Fabricación de lápices, bolígrafos, sellos y artículos de librería en general",
	ActEcon32902: "Fabricación de escobas, cepillos, pinceles y similares",
	ActEcon32903: "Fabricación de artesanías de materiales diversos",
	ActEcon32904: "Fabricación de artículos de uso personal y domésticos n.c.p.",
}

// GetEconomicActivityName returns the name of an economic activity by code
func GetEconomicActivityName(code string) (string, bool) {
	name, exists := EconomicActivities[code]
	return name, exists
}

// GetEconomicActivityCode returns the code for an economic activity by name (case-insensitive)
func GetEconomicActivityCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for code, value := range EconomicActivities {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllEconomicActivities returns a slice of all economic activities
func GetAllEconomicActivities() []EconomicActivity {
	activities := make([]EconomicActivity, 0, len(EconomicActivities))
	for code, value := range EconomicActivities {
		activities = append(activities, EconomicActivity{
			Code:  code,
			Value: value,
		})
	}
	return activities
}

// IsValidEconomicActivity checks if an economic activity code is valid
func IsValidEconomicActivity(code string) bool {
	_, exists := EconomicActivities[code]
	return exists
}
