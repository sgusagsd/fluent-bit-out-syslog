#!/bin/sh
# -*- coding: utf-8 -*-
# Copyright © 2014 VMware, Inc.  All rights reserved.
# VMware Confidential
#
# This script is to inventory the package used in a virtual appliance.
# It should be copied to a virtual appliance, then be executed.
#

ISSUE_FILE=/etc/issue

########
# Help #
########
#
# Print help information on the command and exit.
#

Help () {
    Usage
    cat <<EOF

Inventory the packages installed on a VM.  This utility should be copied to
the target Virtual Machine and executed.  The generated output (osstpmgt.yaml
by default) can then be loaded into the OSM system using the osstp-load
utility.  The OS-NAME is a required argument, e.g., 'photon', 'ubuntu',
'sles', 'centos', etc.

Generally, only the command line operating system name is needed, e.g.,

    $ vm-inventory.sh photon

See https://osm.eng.vmware.com/doc-topic/osstp-utilities-vapp/ for more details.

optional arguments:
  -h, --help            show this help message and exit
  -m GENERATED-MANIFEST Name of the manifest file to generate (osstpmgt.yaml)
  -s OS-STYLE           Operating system style (rpm or deb)
EOF
    exit
}

#########
# Usage #
#########
#
# Print a usage message.  Since this is used by the Help routine, the caller
# needed to exit.
#

Usage() {
    cat <<EOF
usage: vm-inventory [-h|--help] [-m GENERATED-MANIFEST] [-s OS-STYLE] OS-NAME
EOF
}

#########
# Error #
#########
#
# Print an error message and exit.
#

Error() {
    echo "Error: $*" 1>&2
    exit 1
}

##############
# GuessStyle #
##############
#
# Guess the packaging style on the current host based on the /etc/issue file
# and, failing that, standard control directories.
#

GuessStyle() {
    if [ -r $ISSUE_FILE ]
    then
        grep -i -q centos $ISSUE_FILE
        [ $? -eq 0 ] && { Style=rpm; return; }

        grep -i -q ubuntu $ISSUE_FILE
        [ $? -eq 0 ] && { Style=deb; return; }
    fi

    [ -d /var/lib/rpm ] && { Style=rpm; return; }
    [ -d /var/lib/apt ] && { Style=deb; return; }

    Error "Could not determine packaging style, please use the -s option"
}

################
# RpmInventory #
################
#
# Generate the OSSTP manifest based on an RPM based packaging system, e.g.,
# CentOS, SLES, etc.
#

RpmInventory() {
    Manifest=$1
    OSName=$2
    echo "Inventorying a host using .rpm based packaging system"

    # First determine if the host system allows RPM querying of the release
    # (CentOS does, SLES does not).  If the release tag is available, use it
    # in the RPM query string, if not, use an empty value.
    rpm --querytags | grep -i -q '^release$'
    if [ $? -eq 0 ]
    then
        echo "This host supports the \"RELEASE\" RPM tag, manifest will include it"
        RpmQueryFormat="%{name}:%{version}:%{release}:%{sourcerpm}:$OSName\n"
    else
        echo "This host does not support the RELEASE RPM tag, ignore it"
        RpmQueryFormat="%{name}:%{version}::%{sourcerpm}:$OSName\n"
    fi

    # Now query the installed RPM's and format as a OSSTP manifest file
    rpm -qa --qf $RpmQueryFormat | \
        sort | \
        uniq | \
        awk -F: '{
            printf("baseos:rpm:%s:%s:%s:%s:\n", $1, $2, $3, $5);
            printf("    repository: BaseOS\n");
            printf("    name: %c%s%c\n", 39, $1, 39);
            printf("    version: %c%s%c\n", 39, $2, 39);
            printf("    baseos-style: rpm\n");
            printf("    baseos-source: %c%s%c\n", 39, $4, 39);
            printf("    baseos-osname: %c%s%c\n", 39, $5, 39);
            if ($3) {
                printf("    baseos-release: %c%s%c\n", 39, $3, 39);
            }
        }' > $Manifest
}

################
# DebInventory #
################
#
# Generate the OSSTP manifest based on an debian based packaging system
# e.g., Ubuntu.
#

DebInventory() {
    Manifest=$1
    OSName=$2
    echo "Inventorying a host using .deb based packaging system"
    # Run dpkg-query to inventory the packages
    dpkg-query -W -f="\${Package}#\${Version}#\${Source}#$OSName\n" | \
        sort | \
        uniq | \
        awk -F\# '{
            printf("baseos:deb:%s:%s:%s:\n", $1, $2, $4);
            printf("    repository: BaseOS\n");
            printf("    name: %c%s%c\n", 39, $1, 39);
            printf("    version: %c%s%c\n", 39, $2, 39);
            printf("    baseos-style: deb\n");
            printf("    baseos-osname: %c%s%c\n", 39, $4, 39);
            if ($3) {
                printf("    baseos-source: %c%s%c\n", 39, $3, 39);
            }
        }' > $Manifest
}

# Process the command line arguments
Manifest=osstpmgt.yaml
Style=
OSName=
while [ $# -gt 0 ]
do
    arg=$1; shift
    case "$arg" in
    -h|--help)
        Help
        ;;
    -m)
        if [ $# -gt 0 ]
        then
            Manifest=$1; shift
        else
            Usage
            exit 1
        fi
        ;;
    -s)
        if [ $# -gt 0 ]
        then
            Style=$1; shift
        else
            Usage
            exit 1
        fi
        ;;
    -*)
        Usage
        exit 1
        ;;
    *)
        if [ -z "$OSname" ]
        then
            OSName=$arg
        else
            Error "OS name should only be specified once"
        fi
        ;;
    esac
done

# Check that an OS name has been specified
[ -z "$OSName" ] && Error "Please specify an OS name, e.g., 'sles', 'centos'"

# Apply default style if the user didn't specify one
[ -z "$Style" ] && GuessStyle

# Clean up from a previous execution
if [ -r $Manifest ]
then
    echo "Removing old \"$Manifest\" OSM manifest file"
    rm -f $Manifest
fi

# OK, based on the style, generate the installed package inventory
case "$Style" in
rpm)
    RpmInventory $Manifest $OSName
    ;;
deb)
    DebInventory $Manifest $OSName
    ;;
*)
    Error "Unsupported packaging style \"$Style\""
    ;;
esac

NPackages=`grep '^b' $Manifest | wc -l`
echo "Generated the OSM manifest file \"$Manifest\" with $NPackages entries"

exit
