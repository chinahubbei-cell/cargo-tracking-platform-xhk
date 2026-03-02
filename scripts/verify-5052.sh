#!/bin/bash
set -e
curl -s -o /dev/null -w '5052_health:%{http_code}\n' http://127.0.0.1:5052/api/health
curl -s -o /dev/null -w '5180_login:%{http_code}\n' http://127.0.0.1:5180/login
curl -s -o /dev/null -w '5181_login:%{http_code}\n' http://127.0.0.1:5181/login
curl -s -o /dev/null -w '5180_proxy_api:%{http_code}\n' http://127.0.0.1:5180/api/health
curl -s -o /dev/null -w '5181_proxy_api:%{http_code}\n' http://127.0.0.1:5181/api/health
