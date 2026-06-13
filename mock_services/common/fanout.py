from concurrent.futures import ThreadPoolExecutor, as_completed


def parallel(callables):
    """Run named callables concurrently; returns {name: result}."""
    if not callables:
        return {}

    # Per-request pool: a module-level executor is shut down when gunicorn
    # recycles workers (SIGTERM, timeout, compose rebuild) but the worker
    # process can still accept traffic — causing "cannot schedule new futures
    # after shutdown".
    workers = min(32, len(callables))
    with ThreadPoolExecutor(max_workers=workers) as pool:
        futures = {pool.submit(fn): name for name, fn in callables.items()}
        results = {}
        for future in as_completed(futures):
            results[futures[future]] = future.result()
        return results
