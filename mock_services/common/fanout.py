from concurrent.futures import ThreadPoolExecutor, as_completed

_pool = ThreadPoolExecutor(max_workers=32)


def parallel(callables):
    """Run named callables concurrently; returns {name: result}."""
    futures = {_pool.submit(fn): name for name, fn in callables.items()}
    results = {}
    for future in as_completed(futures):
        results[futures[future]] = future.result()
    return results
