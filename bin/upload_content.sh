#!/bin/bash

# Sync to production server from local content folder.

rsync --delete -azvvq -e ssh ./content/ chez@chezwatts.gallery:/home/chez/website-content/