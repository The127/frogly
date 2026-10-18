# a project of your own that takes requests
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl club", "visibility": "public", "joining": "request"}
HTTP 201
[Captures]
id: jsonpath "$.id"
[Asserts]
jsonpath "$.joining" == "request"

# asking as a member changes nothing
POST {{frogly}}/api/projects/{{id}}/requests
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$.asked" == true

# so nobody waits
GET {{frogly}}/api/projects/{{id}}/requests
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
header "Cache-Control" == "no-store"
jsonpath "$" isCollection
jsonpath "$" isEmpty

# a project that doesn't exist can't be asked
POST {{frogly}}/api/projects/01a14180-d957-7cc5-af29-7f1fe18d89bb/requests
Authorization: Bearer {{token}}
HTTP 404
