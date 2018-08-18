docker run -d --rm -p 27017:27017 mongo:4.1.2-xenial


validacoes em cima da propria estrutura, acho que deveriam estar no objeto, nao? no servico entram as coordenacoes de negocio


global monitoring como o caddy fez.
envia as metricas para um servidor central.
como manter compatibilidade com novas versoes?


https://github.com/hadolint/hadolint
https://fossa.io/
https://codebeat.co/projects/github-com-diegobernardes-flare-master/all_complexities
https://github.com/marketplace/codacy

> ci
https://github.com/marketplace/semaphore
https://github.com/marketplace/travis-ci
https://github.com/marketplace/appveyor (windows)

> cover
https://github.com/marketplace/coveralls
https://codecov.io/gh/diegobernardes/flare

> bot
https://probot.github.io/apps/stale/
https://probot.github.io/apps/dco/
https://probot.github.io/apps/welcome/
https://probot.github.io/apps/first-timers/
https://probot.github.io/apps/commitlint/ (yeaaaa)
https://probot.github.io/apps/delete-merged-branch/
https://probot.github.io/apps/triage-new-issues/
https://probot.github.io/apps/semantic-pull-requests/ (maybe)
https://probot.github.io/apps/pr-triage/
https://github.com/jusx/mergeable#configuration
https://probot.github.io/apps/linter-alex/
https://probot.github.io/apps/issuelabeler/
https://probot.github.io/apps/minimum-reviews/
https://probot.github.io/apps/issue-complete/
https://probot.github.io/apps/auto-assign/

https://github.com/search?q=topic%3Aprobot-app&type=Repositories



como vamos fazer o processo do tenant?

usuario entra no site e se cadastra, logo, ele ganha um subdominio:

worten.flarehq.com
authentication: xyz

vamos ter a aplicação que controla algumas coisas, e depois vamos ter a api.
a aplicação, pode ser algo em rails, a api vai ser em go.
futuramente, podemos expor mais dados na api e colocar tudo como single page application.


sonae.flarehq.com/tenants
sonae.flarehq.com/tenants/worten

sonae.worten.flarehq.com/resources
sonae.worten.flarehq.com/resources/1/subscriptions
sonae.worten.flarehq.com/documents

/sonae/worten/tenants/backend
/sonae/worten/backend/resources
