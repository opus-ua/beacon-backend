cp beacon /usr/bin/beacon
cp ./beacon.service /etc/systemd/system/beacon.service
cp ./beacon-dev.service /etc/systemd/system/beacon-dev.service
systemctl daemon-reload
systemctl enable beacon
systemctl start beacon
