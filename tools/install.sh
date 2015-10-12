cp beacon /usr/bin/beacon
cp ./beacon.service /etc/systemd/system/beacon.service
systemctl daemon-reload
systemctl enable beacon
systemctl start beacon
