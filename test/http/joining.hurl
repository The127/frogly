# a public project to open and close
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl fair", "visibility": "public", "joining": "closed"}
HTTP 201
[Captures]
id: jsonpath "$.id"

# a member opens it and gets the project back
PUT {{frogly}}/api/projects/{{id}}/joining
Authorization: Bearer {{token}}
Content-Type: application/json
{"joining": "open"}
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$.id" == "{{id}}"
jsonpath "$.joining" == "open"

# a setting frogly doesn't know says what's wrong
PUT {{frogly}}/api/projects/{{id}}/joining
Authorization: Bearer {{token}}
Content-Type: application/json
{"joining": "maybe"}
HTTP 400
[Asserts]
header "Content-Type" == "application/problem+json"
jsonpath "$.detail" == "a project is open to join, takes requests or is closed"

# a project that doesn't exist is not found
PUT {{frogly}}/api/projects/01a14180-d957-7cc5-af29-7f1fe18d89bb/joining
Authorization: Bearer {{token}}
Content-Type: application/json
{"joining": "open"}
HTTP 404
