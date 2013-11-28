// Yes, it's pathetic, but I can't program JS without SO.

// http://stackoverflow.com/questions/1860524/javascript-how-to-find-out-width-and-height-of-viewpoint-in-browser-window#1860596

function getViewPortWidth()
{
    var viewportwidth;

    //Standards compliant browsers (mozilla/netscape/opera/IE7)
    if (typeof window.innerWidth != 'undefined')
    {
    	viewportwidth = window.innerWidth
    }

    // IE6
    else if (typeof document.documentElement != 'undefined'
    && typeof document.documentElement.clientWidth !=
    'undefined' && document.documentElement.clientWidth != 0)
    {
    	viewportwidth = document.documentElement.clientWidth
    }

    //Older IE
    else
    {
    	viewportwidth = document.getElementsByTagName('body')[0].clientWidth
    }

    return viewportwidth
}

// http://stackoverflow.com/questions/260749/what-is-the-best-way-to-get-and-set-a-single-cookie-value-using-javascript#260880
function createCookie(name,value,days) {
    if (days) {
        var date = new Date();
        date.setTime(date.getTime()+(days*24*60*60*1000));
        var expires = "; expires="+date.toGMTString();
    }
    else var expires = "";
    document.cookie = name+"="+value+expires+"; path=/";
}

// send our width in to the server on the next request so that it can
// tailor the image sizes to our screen size.
var w = getViewPortWidth()
createCookie("w", w, 1)
