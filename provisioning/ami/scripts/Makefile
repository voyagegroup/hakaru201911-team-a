# provisioning target (run only in ami building)
#
# パッケージのインストール等はここでやる
#
.PHONY: all cloud-config yum app

all: cloud-config yum app /etc/security/limits.conf

cloud-config: cloud.cfg.d/99_defaults.cfg
	cp cloud.cfg.d/99_defaults.cfg /etc/cloud/cloud.cfg.d/
	chmod 0644 /etc/cloud/cloud.cfg.d/99_defaults.cfg
	chown -R root:root /etc/cloud/cloud.cfg.d/99_defaults.cfg

yum:
	yum -y update
	yum -y install https://dev.mysql.com/get/mysql-community-release-el7-5.noarch.rpm
	yum -y --enablerepo=mysql56-community install mysql-community-client
	yum -y install java-1.8.0-openjdk
	yum -y install https://s3.amazonaws.com/amazoncloudwatch-agent/amazon_linux/amd64/latest/amazon-cloudwatch-agent.rpm
	yum -y install https://s3.amazonaws.com/ec2-downloads-windows/SSMAgent/latest/linux_amd64/amazon-ssm-agent.rpm
	DD_API_KEY=0c0b6fb2234aabe7865f36ea4776d553 bash -c "$$(curl -L https://raw.githubusercontent.com/DataDog/datadog-agent/master/cmd/agent/install_script.sh)"

/root/hakaru:
	mkdir -p /root/hakaru

/root/hakaru/Makefile: Makefile /root/hakaru
	cp Makefile /root/hakaru/Makefile

/etc/security/limits.conf: limits.conf
	cp limits.conf /etc/security/limits.conf

app: /root/hakaru/Makefile

# application deployment
#
# /root/hakaru/Makefile on ec2 instance
#

ARTIFACTS_BUCKET ?= sunrise201911-team-a-hakaru-artifacts
ARTIFACTS_COMMIT ?= latest

deploy: clean /root/hakaru/app
	$(MAKE) -C /root/hakaru/app/provisioning/instance

clean:
	rm -rf /tmp/artifacts.tgz /root/hakaru/app

/tmp/artifacts.tgz:
	aws s3 cp s3://$(ARTIFACTS_BUCKET)/$(ARTIFACTS_COMMIT)/artifacts.tgz /tmp/artifacts.tgz

/root/hakaru/app: /tmp/artifacts.tgz
	mkdir -p /root/hakaru/app
	tar xzvf /tmp/artifacts.tgz -C /root/hakaru/app
