#!/bin/bash

# Sync to local content folder from production server.

rsync --delete -azvvq -e ssh chez@chezwatts.gallery:/home/chez/website-content/ ./content/