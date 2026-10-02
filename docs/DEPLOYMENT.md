# Deployment

Ubuntu/Debian + aaPanel compatible, no Docker required. PostgreSQL 14+,
Redis 6+. systemd units in `deployments/systemd`. Reverse proxy: nginx
example in `deployments/nginx`. TLS via certbot. Workers: `ispctl worker`,
scheduler: `ispctl scheduler`, RADIUS ports 1812/1813.
