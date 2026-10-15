# every page outside the API is the frontend's, it decides what to show
GET {{frogly}}/
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
header "Cache-Control" == "no-cache"
body contains "<div id=\"app\"></div>"

# Keyline sends the person back here after the login
GET {{frogly}}/callback?code=x&state=y
HTTP 200
[Asserts]
body contains "<div id=\"app\"></div>"
