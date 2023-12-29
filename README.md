[![Automatic Tests]([github-action-automatic-tests-shild])]([github-action-automatic-tests-url)
[![Test Build][github-action-shield]][github-action-url]
[![Go Report Card][reportcard-shield]][reportcard-url]
[![Issues][issues-shield]][issues-url]
[![Issues][stars-shield]][stars-url]

<p align="center">
  <h2 align="center">Arrower</h2>

  <p align="center">
    Arrows to hit your development needs.
    <br />
    <br />
    <a href="https://github.com/go-arrower/arrower#about-the-project">Why?</a>
    ·
    <a href="https://github.com/go-arrower/arrower/issues">Report Bug</a>
    ·
    <a href="https://github.com/go-arrower/arrower/issues">Request Feature</a>
  </p>
</p>




## About the Project
A complete framework to develop Go web applications.

Learn more about the [motivation](https://www.arrower.org/docs/why) behind arrower.


## Create new Database Migration
```shell
export POSTGRESQL_URL='postgres://arrower:secret@localhost:5432/arrower?sslmode=disable'
migrate create -ext sql -dir postgres/migrations -seq create_test_table

migrate -database ${POSTGRESQL_URL} -path postgres/migrations up
migrate -database ${POSTGRESQL_URL} -path postgres/migrations down
```




<!-- MARKDOWN LINKS & IMAGES -->
[github-action-automatic-tests-shild]: https://github.com/go-arrower/arrower/actions/workflows/automatic-tests.yml/badge.svg
[github-action-automatic-tests-url]: https://github.com/go-arrower/arrower/actions/workflows/automatic-tests.yml
[github-action-shield]: https://github.com/go-arrower/arrower/actions/workflows/test.yml/badge.svg
[github-action-url]: https://github.com/go-arrower/arrower/actions
[reportcard-shield]: https://goreportcard.com/badge/github.com/go-arrower/arrower
[reportcard-url]: https://goreportcard.com/report/github.com/go-arrower/arrower
[issues-shield]: https://img.shields.io/github/issues/go-arrower/arrower?style=flat-square&logo=appveyor
[issues-url]: https://github.com/go-arrower/arrower/issues
[stars-shield]: https://img.shields.io/github/stars/go-arrower/arrower?style=flat-square&logo=appveyor
[stars-url]: https://github.com/go-arrower/arrower/stargazers
