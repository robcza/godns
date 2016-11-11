#!/bin/bash

# DEBUG | INFO |NOTICE | WARN | ERROR
# string
export SINKIT_LOG_LEVEL="DEBUG"

# Logging file, absolute or relative
# string
export SINKIT_LOG_FILE="./godns.log"

# Logging, whether we log also to std out
# bool
export SINKIT_LOG_STDOUT="true"

# Address GoDNS proxy should listen on
# string
export SINKIT_BIND_HOST="127.0.0.1"

# Port GoDNS proxy should listen on
# int
export SINKIT_BIND_PORT=5551

# How many CPU cores should we utilize?
# int, 0 means runtime.NumCPU()
export SINKIT_NUM_OF_CPUS=0



# GoDNS server lookup r/w timeout for queries
# int, milliseconds
export SINKIT_GODNS_READ_TIMEOUT=5000
export SINKIT_GODNS_WRITE_TIMEOUT=5000

# resolv.conf file, source of additional resolvers
# string
export SINKIT_RESOLV_CONF_FILE="/etc/resolv.conf"

# Timeout for getting response from backend recursive resolver
# int, milliseconds
export SINKIT_BACKEND_RESOLVER_RW_TIMEOUT=5000

# Tick between trying different backend recursive resolvers
# int, milliseconds
export SINKIT_BACKEND_RESOLVER_TICK=20

# Backend recursive dns resolvers added before resolv.conf
# string, comma separated list of IP:PORT
export SINKIT_BACKEND_RESOLVERS="8.8.8.8:53"

# Backend recursive dns resolvers from SINKIT_BACKEND_RESOLVERS are the
# only resolvers to be used, SINKIT_RESOLV_CONF_FILE is ignored.
# bool
export SINKIT_BACKEND_RESOLVERS_EXCLUSIVELY="true"



# Oraculum responses cache backend
# string, "memory" is the only on implemented
export SINKIT_ORACULUM_CACHE_BACKEND="memory"

# Oraculum responses cache record expiration
# int, milliseconds
export SINKIT_ORACULUM_CACHE_EXPIRE=5000

# Oraculum responses cache maximum records
# int, 0 means the sum of cache items will be unlimited
export SINKIT_ORACULUM_CACHE_MAXCOUNT=0

# Oraculum API fit response time
# int, milliseconds
export SINKIT_ORACULUM_API_FIT_TIMEOUT=500

# Oraculum API hard timeout on HTTP request
# int, milliseconds
export SINKIT_ORACULUM_API_TIMEOUT=600

# Oraculum should not be contacted after failure for some time.
# int, milliseconds
export SINKIT_ORACULUM_SLEEP_WHEN_DISABLED=20000

# Oraculum requests could be explicitly disabled in the configuration.
# bool
export SINKIT_ORACULUM_DISABLED="false"

# Oraculum REST API URL
# string
export SINKIT_ORACULUM_URL="http://127.0.0.1:8080/sinkit/rest/blacklist/dns"

# Oraculum API access token, header key and header value
# string
export SINKIT_ORACULUM_ACCESS_TOKEN_KEY="X-sinkit-token"
export SINKIT_ORACULUM_ACCESS_TOKEN_VALUE="kjdqgkjhgdajdsakgqq"

# Sinkhole address. We don't use the one returned from
# Oraculum at the moment.
export SINKIT_SINKHOLE_ADDRESS="127.0.0.1"

# Skip IP address calls to Oraculum REST API
# bool
export SINKIT_ORACULUM_IP_ADDRESSES_ENABLED="false"
