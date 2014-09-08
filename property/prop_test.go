package property

import (
	"reflect"
	"strings"
	"testing"
)

func TestTrim(t *testing.T) {
	if trimM("8'001") != "8001" {
		t.Error("failed")
	}
	if trimM("8'001 m") != "8001" {
		t.Error("failed")
	}
}

func TestReader(t *testing.T) {
	r := strings.NewReader(in108)
	p, err := Read(r)
	if err != nil {
		t.Error("unexpected error: ", err)
	}
	exp := &Property{
		Commune: "Mont-la-Ville",
		Id:      108,
		Surface: 821,
		Owners:  []string{"Allen Jeffrey", "Canepa Allen Marina"},
	}
	if !reflect.DeepEqual(p, exp) {
		t.Error("not equal: %v != %v", p, exp)
	}
}

const in108 = `
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=ISO-8859-1">
<title>Registre foncier</title>
</head>
<body><font face="Arial"><font face="arial" size="1">Commune : </font><font face="arial, Helvetica, sans-serif" size="1" color="#0033ff">Mont-la-Ville</font><br><font face="arial" size="1">Bien-fonds : </font><font face="arial, Helvetica, sans-serif" size="1" color="#0033ff">108</font><br><hr>
<table cellspacing="0" cellpadding="1">
<tr><td colspan="3" align="left"><font face="arial, Helvetica, sans-serif" size="1" color="#ff0000">Surface : </font></td></tr>
<tr><td colspan="3" align="right" bgcolor="#efefef"><font face="arial, Helvetica, sans-serif" size="1">821 m<sup>2</sup></font></td></tr>
<tr><td colspan="3" height="10"></td></tr>
<tr><td colspan="3" align="left"><font face="arial, Helvetica, sans-serif" size="1" color="#ff0000">Genre(s) de nature et bâtiment(s) : </font></td></tr>
<tr>
<td align="left" valign="bottom" bgcolor="#efefef"><font face="arial, Helvetica, sans-serif" size="1">Habitation</font></td>
<td width="5px" bgcolor="#efefef"><font face="arial, Helvetica, sans-serif" size="1"> </font></td>
<td align="right" valign="bottom" bgcolor="#efefef"><font face="arial, Helvetica, sans-serif" size="1">83 m<sup>2</sup></font></td>
</tr>
<tr>
<td align="left" valign="bottom" bgcolor="#ffffff"><font face="arial, Helvetica, sans-serif" size="1">Jardin</font></td>
<td width="5px" bgcolor="#ffffff"><font face="arial, Helvetica, sans-serif" size="1"> </font></td>
<td align="right" valign="bottom" bgcolor="#ffffff"><font face="arial, Helvetica, sans-serif" size="1">738 m<sup>2</sup></font></td>
</tr>
<tr><td colspan="3" height="10"></td></tr>
<tr><td colspan="3" align="left"><font face="arial, Helvetica, sans-serif" size="1" color="#ff0000">Propriétaire(s) : </font></td></tr>
<tr><td colspan="3" align="left" bgcolor="#efefef"><font face="arial, Helvetica, sans-serif" size="1">Allen Jeffrey</font></td></tr>
<tr><td colspan="3" align="left" bgcolor="#ffffff"><font face="arial, Helvetica, sans-serif" size="1">Canepa Allen Marina</font></td></tr>
</table>
<br><font face="arial" size="1"> <a href="http://www.rf.vd.ch" target="_blank">Registre Foncier</a> (Service payant)</font><br><br><font face="arial" size="1"> Note : la surface des bâtiments souterrains et des couverts n'est pas comptabilisée dans la surface totale du bien-fonds.</font><br><br><font face="Arial" size="1">Interop RF version 1.5</font></font></body>
</html>
`
