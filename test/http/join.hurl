# a project of your own, open to join
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl fair", "visibility": "public", "joining": "open"}
HTTP 201
[Captures]
open: jsonpath "$.id"
[Asserts]
jsonpath "$.joining" == "open"

# joining a project you're in changes nothing and answers with it
POST {{frogly}}/api/projects/{{open}}/join
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$.id" == "{{open}}"

# a project that doesn't exist can't be joined
POST {{frogly}}/api/projects/01a14180-d957-7cc5-af29-7f1fe18d89bb/join
Authorization: Bearer {{token}}
HTTP 404
