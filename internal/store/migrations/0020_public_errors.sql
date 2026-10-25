-- Failed jobs stored the raw error, which could name files or the storage backend; the API shows it to users and shares.
UPDATE coverages SET job_error = 'Calculation failed.' WHERE job_error IS NOT NULL AND job_error <> '';
