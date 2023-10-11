# Test Data
The [testdata directory](./) is split into 2 subdirectories.  
- [localcnudierepos](./localcnudierepos) contains multiple local repositories with component descriptors v2 (also known 
as cnudie component descriptors). These can be used by both facade implementations, [cnudie](../cnudie) and 
[ocmlib](../ocmlib). 
- [localocmrepos](./localocmrepos) contains essentially the same local repositories with the same components but
with component descriptors v3 (also known as ocm component descriptors). These can only be used by the 
[ocmlib](../ocmlib) facade implementation.

As the [localcnudierepos](./localcnudierepos) and the [localocmrepos](./localocmrepos) are essentially mirroring each
other with the only difference of the serialization format (component descriptors v2 vs component descriptors v3), these
can be used if the behavior of both facade implementation is the same if they both use the component descriptors v2 and
also if the behavior of the [ocmlib](../ocmlib) is the same with both component descriptor versions.

