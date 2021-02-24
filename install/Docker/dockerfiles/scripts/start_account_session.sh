#!/bin/bash
# (C) Copyright [2020] Hewlett Packard Enterprise Development LP
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License. You may obtain
# a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations
# under the License.

declare PID=0
declare OWN_PID=$$

sigterm_handler()
{
        if [[ $PID -ne 0 ]]; then
                sleep 1
                kill -9 $PID
                wait "$PID" 2>/dev/null
        fi
        exit 0
}

# create a signal trap
create_signal_trap()
{
        trap 'echo "[$(date)] -- INFO  -- SIGTERM received for account_session, initiating shut down"; sigterm_handler' SIGTERM
}

# keep the script running till SIGTERM is received
run_forever()
{
        wait
}

start_account_session()
{
	registry_address="consul:8500"
	if [[ ${HA_ENABLED,,} == true ]]; then
		if [[ -z "${ODIM_NAMESPACE}" ]]; then
			echo "[$(date)] -- ERROR -- ODIM_NAMESPACE variable not set, exiting"
			exit 1
		fi
		consul_addr_suffix="consul.${ODIM_NAMESPACE}.svc.cluster.local:8500"
		registry_address="consul1.${consul_addr_suffix},consul2.${consul_addr_suffix},consul3.${consul_addr_suffix}"
	fi

	export CONFIG_FILE_PATH=/etc/odimra_config/odimra_config.json
	nohup /bin/svc-account-session --registry=consul --registry_address=${registry_address} --server_address=account-session:45101 --client_request_timeout=`expr $(cat $CONFIG_FILE_PATH | grep SouthBoundRequestTimeoutInSecs | cut -d : -f2 | cut -d , -f1 | tr -d " ")`s >> /var/log/odimra_logs/account_session.log 2>&1 &
	PID=$!
	sleep 3

	nohup /bin/add-hosts -file /tmp/host.append >> /var/log/odimra_logs/account-session-add-hosts.log 2>&1 &
}

monitor_process()
{
        while true; do
                pid=$(pgrep -fc svc-account-session 2> /dev/null)
                if [[ $? -ne 0 ]] || [[ $pid -gt 1 ]]; then
                        echo "[$(date)] -- ERROR -- svc-account-session not found running, exiting"
			kill -15 ${OWN_PID}
			exit 1
                fi
                sleep 5
        done &
}

##############################################
###############  MAIN  #######################
##############################################

start_account_session

create_signal_trap

monitor_process

run_forever

exit 0
