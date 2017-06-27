#!/bin/bash
# @author Michal Karm Babacek

export NUM_OF_MY_CPUS="`nproc`"
export UNBOUND_CONFIG_FILE="/etc/unbound/unbound.conf"

# Credit: John1024, SO
pw2() { echo "x=l($1)/l(2); scale=0; 2^((x+0.5)/1)" | bc -l; }

if [[ "${NUM_OF_MY_CPUS}" -lt 2 ]]; then
    echo "Too few CPUs. Use at least 2 cores, recommended 8."
    exit 1
fi

if [[ "${NUM_OF_MY_CPUS}" -le 4 ]]; then
    echo "With just 4 CPUs, we consider this box to be a test machine; using all cores."
    export UNBOUND_NUM_THREADS=${NUM_OF_MY_CPUS}
    export SINKIT_NUM_OF_CPUS=${NUM_OF_MY_CPUS}
else
    # If the number is odd, the +1 goes to GoDNS
    export UNBOUND_NUM_THREADS=$((${NUM_OF_MY_CPUS}/2))
    export SINKIT_NUM_OF_CPUS=$((${NUM_OF_MY_CPUS}/2 + (${NUM_OF_MY_CPUS} % 2)))
fi

UB_SLABS=`pw2 ${UNBOUND_NUM_THREADS}`
export UNBOUND_INFRA_CACHE_SLABS=${UB_SLABS}
export UNBOUND_RRSET_CACHE_SLABS=${UB_SLABS}
export UNBOUND_MSG_CACHE_SLABS=${UB_SLABS}

export TOTAL_RAM_K=$(free -k | awk '/^Mem:/{print $2}')
export TOTAL_RAM_B=$(($TOTAL_RAM_K*1024))

# Unbound should use up to n % of total RAM
UNBOUND_MAX_PERCENT=${UNBOUND_MAX_PERCENT:-30}

# GoDNS should use up to n % of total RAM
GODNS_MAX_PERCENT=${GODNS_MAX_PERCENT:-30}

# GoDNS oraculum
HOW_MUCH_ONE_CACHE_RECORD_TAKES_B=1000
export SINKIT_ORACULUM_CACHE_MAXCOUNT=`echo "scale=0; ($TOTAL_RAM_B/100*$GODNS_MAX_PERCENT)/$HOW_MUCH_ONE_CACHE_RECORD_TAKES_B;" | bc`

UB_RAM_K=`echo "scale=0; $TOTAL_RAM_K/100*$UNBOUND_MAX_PERCENT;" | bc`

# See the Unbound doc, mind malloc overhead
export UNBOUND_RRSET_CACHE_SIZE="$(($UB_RAM_K/2))k"
export UNBOUND_MSG_CACHE_SIZE="$(($UB_RAM_K/4))k"

# Hard limits that trigger process restart
# If either of these eats up to:
HARD_MEM_LIMIT_PERCENT=${HARD_MEM_LIMIT_PERCENT:-60}
export GODNS_HARD_RAM_LIMIT="`echo "scale=0; $TOTAL_RAM_K/100*$HARD_MEM_LIMIT_PERCENT;" | bc`KB"
export UNBOUND_HARD_RAM_LIMIT=${GODNS_HARD_RAM_LIMIT}

# Start
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf -n
