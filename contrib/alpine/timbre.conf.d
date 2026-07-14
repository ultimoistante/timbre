# Sourced automatically by /etc/init.d/timbre (OpenRC conf.d convention).
# Install: cp to /etc/conf.d/timbre.

TIMBRE_DATA_DIR="/var/lib/timbre"
TIMBRE_HOST="0.0.0.0"
TIMBRE_PORT="8080"
TIMBRE_DB_DRIVER="sqlite"

# PostgreSQL instead of the sqlite default:
#TIMBRE_DB_DRIVER="postgres"
#TIMBRE_DB_DSN="host=localhost user=timbre password=secret dbname=timbre sslmode=disable"

#TIMBRE_ACCESS_TTL_MIN="30"
#TIMBRE_REFRESH_TTL_DAYS="30"
