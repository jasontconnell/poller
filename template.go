package main

const templateString string = `
{{ define "Results" }}
<html>
    <head>
        <meta http-equiv="refresh" content="15">
        <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/css/bootstrap.min.css" integrity="sha384-Gn5384xqQ1aoWXA+058RXPxPg6fy4IWvTNh0E263XmFcJlSAwiGgFAW/dAiS6JXm" crossorigin="anonymous">
    </head>
    <body>
        <div class="container">
            <table class="table table-hover">
                <thead>
                    <tr>
                        <th scope="col">Url</th>
                        <th scope="col">Status</th>
                        <th scope="col">Next Retry</th>
                        <th scope="col">Last Retry</th>
                    </tr>
                </thead>
                <tbody>
                    {{ range .Items }}
                    <tr class="{{ if not .Success }}table-danger{{end}}">
                        <td>{{.Url}}</td>
                        <td>{{.StatusText}}</td>
                        <td>{{.Retry}}</td>
                        <td>{{.Last}}</td>
                    </tr>
                    {{ end }}
                </tbody>
            </table>
        </div>
    </body>
</html>
{{ end }}

`
