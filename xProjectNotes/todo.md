# TODO
## General
- [x] add setup of .conf instead of hardcoded paths (and decide lang for .conf)
~~- [] (if other than .json) write parser for .conf
~~- [] configure DB for WAL
~~- [] set up separate DB for contents fit FTS5 for text search
- [] how to solve sudo permissions (if needed)
- [] set up autostart
- [] how to manage index status (log file(?) with 'last fresh index', 'last index sync' et.c.)
- [] set up robust error handling/logging
- [] explore options for defining file types for content reading
- [] explore options for defining excluded objects


# BUG FIXES & CHANGES
## General
- [x] change time representations from combined Sec+Nsec to time.Time objects
- [x] fix the multiplied creation of new directories
- [x] store content snippets without regex. only regex full content
~~- [] update db writes to separate metadata and contents
- [] merge the worker management into its own package
- [x] move exclusion- and content selections into config
- [x] optimize bulk db write with statement preparation and commit instead of loop of single row writes
- [] find better method for deletion monitoring
- [] move deletion to its own process due to being to time consuming for the continuous sync
