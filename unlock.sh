#!/bin/sh
restic unlock --cacert config/ca.crt --password-command "echo rpBurwnvmpT3Fh" -r rest:https://bitwarden:rpBurwnvmpT3Fh@restic.3ar.de:8000/bitwarden/
restic unlock --cacert config/ca.crt --password-command "echo YkjD9cmefB4Tbt" -r rest:https://ccfweb:rpBurwnvmpT3Fh@restic.3ar.de:8000/podcast
restic unlock --cacert config/ca.crt --password-command "echo YkjD9cmefB4Tbt" -r rest:https://ccfweb:rpBurwnvmpT3Fh@restic.3ar.de:8000/ccf
restic unlock --cacert config/ca.crt --password-command "echo reknrrgYbmsK8o" -r rest:https://optigem:yfauefie2SdXpL@restic.3ar.de:8000/optigem/
restic unlock --cacert config/ca.crt --password-command "echo W6tkVZvBthCT6d232qZ" -r rest:https://ccfcloud:ioSqSQBYHkf4QUhj6@restic.3ar.de:8000/ccfcloud
