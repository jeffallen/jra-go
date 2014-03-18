package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"code.google.com/p/jra-go/moonrise"
	"github.com/keep94/sunrise"
)

var lat = flag.Float64("lat", 46.647496, "latitude of the camera")
var lon = flag.Float64("lon", 6.40930383, "longitude of the camera")
var nphotos = flag.Int("nphotos", 0, "how many photos to take (debug only)")
var pic = flag.Bool("pic", false, "take a picture now")
var tl = flag.Bool("tl", false, "take a time-lapse now")
var dir = flag.String("dir", ".", "directory to save pictures")
var beforeTL = flag.String("beforeTL", "before-tl", "script to run before time lapse")
var afterTL = flag.String("afterTL", "after-tl", "script to run after time lapse")
var before = flag.Duration("before", 90*time.Minute, "how long to start taking photos before sunrise/sunset")
var period = flag.Duration("period", 45*time.Second, "how long to wait between photos")
var nowdelta = flag.Duration("nowdelta", 0*time.Second, "how far to shift time.Now (debug only)")

func sunriseTime(now time.Time) time.Time {
	ss := &sunrise.Sunrise{}
	ss.Around(*lat, *lon, now)
	return ss.Sunrise()
}

func sunsetTime(now time.Time) time.Time {
	ss := &sunrise.Sunrise{}
	ss.Around(*lat, *lon, now)
	return ss.Sunset()
}

func run(exe string) {
	cmd := exec.Command(exe)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("run error:", err)
	}
}

func now() time.Time {
	return time.Now().Add(*nowdelta)
}

func picture() {
	now := now()
	fn := fmt.Sprintf("%v/%v.jpg", *dir, now.Format("150405"))
	log.Println("photo:", fn)
	cmd := exec.Command("raspistill", "-n", "--width", "1920",
		"--height", "1080", "-ex", "night", "-awb", "horizon", "-o", fn)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("picture error:", err)
	}
}

func timeLapse(count int, sleep time.Duration) {
	run(*beforeTL)
	for i := 0; i < count; i++ {
		// Start a timer, so that the time it takes picture() to run
		// is counted in the sleeping time.
		wake := time.After(sleep)
		picture()
		<-wake
	}
	run(*afterTL)
}

var moonrises = &moonrise.Db{}

func main() {
	flag.Parse()

	moonrises.Load(rpt2014)

	photos := *nphotos
	if photos == 0 {
		// auto: take pictures from sunrise-before until sunrise+before
		photos = int(2 * *before / *period)
	}
	log.Printf("Will take %v pictures, one each %v.", photos, *period)

	// "unit tests": -pic (one picture) or -tl (time lapse)
	if *pic {
		picture()
		return
	}
	if *tl {
		timeLapse(photos, *period)
		return
	}

	for {
		waitForNextEvent()
		timeLapse(photos, *period)
	}
}

func waitForNextEvent() {
	l, _ := time.LoadLocation("Local")
	now := now()

	if *nowdelta != 0*time.Second {
		fmt.Println("fake now is:", now)
	}

	// Calculate times until all the various
	// possible next events here. Then select the
	// next event by finding the minimum.

	what := append([]string{}, "today sunrise")
	s := sunriseTime(now).Add(-*before)
	until := append([]time.Duration{}, s.Sub(now))

	what = append(what, "tomorrow sunrise")
	s = sunriseTime(now.AddDate(0, 0, 1)).Add(-*before)
	until = append(until, s.Sub(now))

	mr, mrok := moonrises.Moonrise(now)
	if mrok {
	what = append(what, "today moonrise")
		until = append(until, mr.Add(-*before).Sub(now))
	}

	mr, mrok = moonrises.Moonrise(now.AddDate(0, 0, 1))
	if mrok {
	what = append(what, "tomorrow moonrise")
		until = append(until, mr.Add(-*before).Sub(now))
	}

	what = append(what, "noon")
	s = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, l)
	s = s.Add(-*before)
	until = append(until, s.Sub(now))

	what = append(what, "today sunset")
	s = sunsetTime(now).Add(-*before)
	until = append(until, s.Sub(now))

	minwhat := "forever?"
	min := time.Duration(99 * time.Hour)

	for i, x := range until {
		// Find minimum durations (ignoring negatives)
		if x >= time.Duration(0) && x < min {
			minwhat = what[i]
			min = x
		}
	}
	log.Println(minwhat, "is at", now.Add(min), ", sleeping", min)
	time.Sleep(min)
}

var rpt2014 = `             o  ,    o  ,                                    MONT-LA-VILLE                             Astronomical Applications Dept.
Location: W006 24, N46 38                         Rise and Set for the Moon for 2014                   U. S. Naval Observatory        
                                                                                                       Washington, DC  20392-5420     
																																																			                                                       Zone:  1h East of Greenwich                                                     
																																																																														                                                                                                                                       
																																																																														                                                                                                                                       
																																																																														        Jan.       Feb.       Mar.       Apr.       May        June       July       Aug.       Sept.      Oct.       Nov.       Dec.  
																																																																																		Day Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set
																																																																																		     h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m
																																																																																				 01  0844 1822  0921 2100  0749 1947  0754 2211  0745 2259  0901 2350  0947 2322  1139 2310  1341 2330  1428       1515 0124  1454 0251
																																																																																				 02  0934 1938  0954 2215  0821 2102  0830 2316  0831 2351  0959       1046 2348  1241 2339  1443       1517 0007  1549 0237  1526 0402
																																																																																				 03  1016 2055  1024 2328  0852 2215  0909       0922       1058 0022  1146       1344       1541 0018  1601 0113  1621 0350  1600 0512
																																																																																				 04  1053 2212  1055       0924 2325  0953 0016  1016 0036  1157 0051  1247 0013  1448 0012  1635 0114  1640 0224  1653 0504  1638 0621
																																																																																				 05  1125 2326  1127 0037  0958       1041 0111  1112 0116  1257 0118  1349 0039  1553 0049  1724 0218  1716 0338  1727 0617  1720 0727
																																																																																				 06  1155       1201 0144  1035 0031  1133 0159  1210 0150  1358 0144  1453 0107  1655 0134  1807 0329  1750 0454  1803 0728  1807 0828
																																																																																				 07  1224 0037  1238 0246  1116 0133  1227 0241  1309 0221  1500 0210  1558 0137  1754 0227  1846 0445  1823 0610  1844 0837  1858 0924
																																																																																				 08  1254 0146  1319 0345  1200 0230  1324 0318  1409 0249  1604 0237  1705 0213  1848 0329  1921 0602  1857 0725  1928 0942  1953 1013
																																																																																				 09  1325 0252  1404 0438  1249 0320  1422 0351  1510 0316  1711 0306  1811 0255  1935 0440  1955 0720  1932 0839  2018 1042  2051 1055
																																																																																				 10  1359 0355  1454 0526  1341 0405  1522 0420  1613 0342  1819 0340  1914 0345  2016 0555  2028 0837  2011 0951  2111 1134  2150 1132
																																																																																				 11  1437 0455  1548 0609  1437 0445  1623 0448  1717 0408  1927 0419  2011 0445  2053 0714  2103 0952  2053 1058  2207 1219  2249 1204
																																																																																				 12  1519 0552  1644 0647  1534 0520  1725 0515  1824 0437  2032 0506  2101 0553  2127 0832  2139 1104  2139 1159  2304 1259  2348 1233
																																																																																				 13  1607 0643  1743 0720  1634 0551  1829 0541  1931 0509  2132 0602  2145 0707  2200 0949  2218 1212  2230 1254       1333       1300
																																																																																				 14  1658 0729  1843 0750  1734 0620  1934 0609  2040 0546  2225 0706  2223 0824  2232 1103  2301 1315  2323 1343  0003 1403  0047 1326
																																																																																				 15  1753 0810  1944 0818  1836 0647  2041 0639  2146 0629  2310 0817  2257 0941  2306 1215  2348 1412       1424  0102 1431  0147 1351
																																																																																				 16  1851 0846  2045 0844  1938 0713  2148 0712  2248 0719  2349 0931  2329 1057  2342 1323       1503  0019 1501  0201 1457  0247 1418
																																																																																				 17  1950 0918  2148 0910  2042 0740  2255 0751  2343 0818       1046       1210       1427  0038 1548  0117 1533  0301 1523  0349 1448
																																																																																				 18  2050 0947  2251 0936  2147 0808  2358 0836       0923  0024 1200  0000 1321  0021 1526  0132 1627  0215 1602  0401 1550  0453 1521
																																																																																				 19  2151 1013  2356 1004  2253 0838       0928  0031 1033  0055 1312  0031 1429  0105 1620  0228 1701  0314 1629  0503 1618  0558 1559
																																																																																				 20  2252 1039       1035  2359 0913  0055 1028  0112 1146  0126 1422  0104 1534  0152 1708  0326 1732  0413 1655  0607 1650  0702 1645
																																																																																				 21  2355 1104  0101 1110       0952  0147 1133  0148 1258  0156 1531  0141 1636  0243 1750  0424 1800  0514 1722  0712 1726  0805 1738
																																																																																				 22       1130  0206 1152  0103 1039  0231 1243  0221 1410  0227 1637  0221 1733  0338 1827  0524 1827  0615 1749  0817 1807  0903 1840
																																																																																				 23  0059 1159  0310 1241  0204 1133  0310 1355  0251 1521  0301 1741  0305 1824  0435 1900  0624 1853  0718 1819  0920 1856  0955 1948
																																																																																				 24  0205 1232  0410 1339  0259 1234  0345 1508  0321 1631  0339 1842  0354 1910  0533 1930  0724 1919  0822 1852  1019 1953  1040 2101
																																																																																				 25  0312 1310  0505 1444  0348 1341  0417 1620  0351 1739  0421 1937  0447 1951  0632 1957  0826 1947  0926 1930  1113 2056  1120 2215
																																																																																				 26  0419 1356  0554 1557  0432 1453  0448 1732  0424 1846  0507 2027  0543 2026  0731 2023  0928 2017  1029 2013  1201 2204  1156 2329
																																																																																				 27  0524 1450  0637 1713  0510 1607  0518 1843  0500 1949  0558 2111  0641 2058  0831 2049  1031 2051  1130 2104  1242 2315  1228     
																																																																																				 28  0625 1555  0714 1830  0545 1722  0550 1952  0539 2049  0653 2150  0739 2126  0932 2115  1134 2130  1226 2201  1319       1259 0042
																																																																																				 29  0719 1707             0617 1837  0625 2059  0624 2143  0750 2224  0838 2153  1033 2143  1236 2215  1316 2305  1352 0027  1331 0153
																																																																																				 30  0805 1824             0648 1950  0703 2202  0713 2231  0848 2254  0938 2218  1135 2214  1334 2308  1400       1423 0139  1403 0303
																																																																																				 31  0846 1942             0720 2102             0806 2313             1038 2244  1238 2249             1440 0013             1439 0411

																																																																																				 NOTE: BLANK SPACES IN THE TABLE INDICATE THAT A RISING OR A SETTING DID NOT OCCUR DURING THAT 24 HR INTERVAL.

																																																																																				 Back to form
`
