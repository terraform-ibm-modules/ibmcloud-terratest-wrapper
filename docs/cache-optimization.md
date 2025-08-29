# API Caching for Permutation Test Optimization

## Overview

The permutation test framework now supports API response caching to dramatically reduce API calls and improve test execution time. This optimization is particularly effective for permutation tests where the same API responses are requested repeatedly across different test combinations.

## Cache Configuration

### Caching is Enabled by Default

Starting with this version, **API caching is enabled by default** for all test options to improve performance. The cache configuration uses pointer types to allow explicit control:

```go
func TestAddonPermutationsWithDefaultCache(t *testing.T) {
    t.Parallel()

    // Cache enabled by default with 10-minute TTL
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing: t,
        Prefix:  "cache-test",
        AddonConfig: &cloudinfo.AddonConfig{
            OfferingName:   "terraform-ibm-observability-agents",
            OfferingFlavor: "standard",
            // Dependencies will be auto-discovered and cached
        },
        // CacheEnabled: defaults to true (enabled)
        // CacheTTL: defaults to 10 minutes
    })

    // Run permutation test - will benefit from caching
    options.RunAddonPermutationTest()

    // Log cache performance after test
    if options.CloudInfoService.IsCacheEnabled() {
        stats := options.CloudInfoService.GetCacheStats()
        t.Logf("Cache performance: %d hits, %d misses, %.1f%% hit rate",
               stats.Hits, stats.Misses, stats.HitRate*100)
    }
}
```

### Explicit Cache Control

To explicitly control caching behavior:

```go
import "github.com/IBM/go-sdk-core/v5/core"

func TestWithExplicitCacheSettings(t *testing.T) {
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:      t,
        Prefix:       "explicit-cache",
        CacheEnabled: core.BoolPtr(false), // Explicitly disable caching
        CacheTTL:     5 * time.Minute,     // Custom TTL (only used if cache enabled)
        AddonConfig: &cloudinfo.AddonConfig{
            OfferingName: "terraform-ibm-observability-agents",
        },
    })

    // Cache is now disabled for this test
    options.RunAddonPermutationTest()
}
```

## Cache Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `CacheEnabled` | `true` (enabled by default) | Enable/disable API response caching |
| `CacheTTL` | `10 minutes` | How long to cache responses |

### Recommended Settings

- **Development/Local Testing**: `CacheTTL: 5 * time.Minute`
- **CI/CD Pipelines**: `CacheTTL: 10 * time.Minute`
- **Long-running Test Suites**: `CacheTTL: 15 * time.Minute`

## Cached API Operations

The following expensive API operations are cached:

### 1. GetOffering Calls
- **Purpose**: Retrieve offering metadata
- **Cache Key**: `catalogID:offeringID`
- **High Impact**: Called for every dependency in every permutation

### 2. GetOfferingVersionLocatorByConstraint
- **Purpose**: Resolve version constraints to specific version locators
- **Cache Key**: `catalogID:offeringID:constraint:flavor`
- **High Impact**: Version resolution for each dependency

### 3. GetCatalogVersionByLocator
- **Purpose**: Get version metadata by locator
- **Cache Key**: `versionLocator`
- **Medium Impact**: Called during dependency validation

### 4. GetComponentReferences (Most Expensive)
- **Purpose**: Get flat list of all dependencies and sub-dependencies
- **Cache Key**: `versionLocator`
- **Highest Impact**: Most expensive API call, called for dependency tree traversal

## Cache Statistics Monitoring

### Programmatic Access

```go
// Get cache statistics
stats := cloudInfoService.GetCacheStats()
fmt.Printf("Offering cache hit rate: %.1f%%\n",
    float64(stats.OfferingHits) / float64(stats.OfferingHits + stats.OfferingMisses) * 100)

// Clear cache if needed
cloudInfoService.ClearCache()

// Check if caching is enabled
if cloudInfoService.IsCacheEnabled() {
    cloudInfoService.LogCacheStats()
}
```


## Best Practices

### 1. Enable Caching for Permutation Tests
Always enable caching when running permutation tests as they benefit most from API call deduplication.

### 2. Monitor Cache Performance
Use `LogCacheStats()` to verify cache effectiveness:
- Target: >70% hit rate for most cached operations
- Monitor API call reduction percentage

### 3. Appropriate TTL Settings
- **Too short**: Reduces cache effectiveness
- **Too long**: May cache stale data (though unlikely in short test runs)
- **Recommended**: 5-15 minutes for most test scenarios

### 4. Clear Cache When Needed
```go
// Clear cache between major test phases if needed
cloudInfoService.ClearCache()
```

### 5. Parallel Test Considerations
- Cache is thread-safe and works with `t.Parallel()`
- Shared cache across parallel tests maximizes efficiency
- Each test process has its own cache instance

## Troubleshooting

### Cache Not Working
1. Verify `CacheEnabled: true` in options
2. Check logs for "API cache enabled" message
3. Ensure `CloudInfoService` is reused across test operations

### Low Hit Rates
1. TTL might be too short for test duration
2. Each unique parameter combination creates new cache entries
3. Check for dynamic/random values in cache keys

### Memory Issues
1. Reduce `CacheTTL` for memory-constrained environments
2. Clear cache periodically for long-running tests
3. Monitor cache statistics for excessive entry counts

## Implementation Details

The caching layer is implemented as an in-memory cache with the following characteristics:

- **Thread-safe**: Uses RWMutex for concurrent access
- **TTL-based expiration**: Configurable time-to-live per entry
- **Memory efficient**: Stores only successful and error responses
- **Statistics tracking**: Comprehensive hit/miss/eviction metrics
- **Zero external dependencies**: Pure Go implementation

This optimization is particularly effective for permutation tests where the same dependency trees are analyzed repeatedly with different enabled/disabled combinations.
