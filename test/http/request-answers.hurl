# a project of your own that takes requests
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl club", "visibility": "public", "joining": "request"}
HTTP 201
[Captures]
id: jsonpath "$.id"

# a yes for someone who never asked is not found
POST {{frogly}}/api/projects/{{id}}/requests/01a14180-d957-7cc5-af29-7f1fe18d89bb/accept
Authorization: Bearer {{token}}
HTTP 404
[Asserts]
header "Content-Type" == "application/problem+json"

# and so is a no
DELETE {{frogly}}/api/projects/{{id}}/requests/01a14180-d957-7cc5-af29-7f1fe18d89bb
Authorization: Bearer {{token}}
HTTP 404
