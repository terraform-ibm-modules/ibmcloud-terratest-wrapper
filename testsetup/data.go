package testsetup

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/crn"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/hashicorp/go-memdb"
)

// the DB record for resources
type CloudResource struct {
	ServiceName string
	Region      string
	Count       int
}

func (res CloudResource) String() string {
	return fmt.Sprintf("Service: %s  |  Region: %s  |  Count: %d", res.ServiceName, res.Region, res.Count)
}

// service limits for region selection
type CloudServiceLimit struct {
	ServiceName      string
	AvailableRegions []string `yaml:",omitempty"`
	ExcludeRegions   []string `yaml:",omitempty"`
	RegionQuota      *int     `yaml:",omitempty"`
}

func (limit CloudServiceLimit) String() string {
	formatQuota := "NOT SET"
	if limit.RegionQuota != nil {
		formatQuota = fmt.Sprintf("%d", *limit.RegionQuota)
	}

	return fmt.Sprintf("Service: %s\n  AvailRegions: %v\n  ExcludeRegions: %v\n  Quota: %s", limit.ServiceName, limit.AvailableRegions, limit.ExcludeRegions, formatQuota)
}

type TestRegionSettings struct {
	AllowedTestRegions []string `yaml:",omitempty"`
}

// creates a copy of a resource record, used for certain DB operations
func (resource *CloudResource) Copy() *CloudResource {
	newResource := new(CloudResource)
	newResource.ServiceName = resource.ServiceName
	newResource.Region = resource.Region
	newResource.Count = resource.Count

	return newResource
}

// creates an empty memdb for resource detail storage
func (svc *TestSetupService) CreateResourceDB() error {

	// Create the DB schema
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"resource": {
				Name: "resource",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:   "id",
						Unique: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "ServiceName"},
								&memdb.StringFieldIndex{Field: "Region"},
							},
							AllowMissing: false,
						},
					},
					"servicename": {
						Name:    "servicename",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "ServiceName"},
					},
					"region": {
						Name:    "region",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Region"},
					},
					"count": {
						Name:    "count",
						Unique:  false,
						Indexer: &memdb.IntFieldIndex{Field: "Count"},
					},
				},
			},
			"limit": {
				Name: "limit",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ServiceName"},
					},
				},
			},
		},
	}

	// Create a new data base
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return err
	}

	svc.ResourceDB = db

	return nil

}

// creates a new CloudResource object using a resource controller instance
// NOTE: CloudResource return could be null with no error, which means we want to skip that resource for some reason
func newCloudResourceRecord(resourceInfo resourcecontrollerv2.ResourceInstance) (*CloudResource, error) {

	var record *CloudResource

	// DEV NOTE: only add resource to DB if CRN exists in record, to avoid any panic
	if resourceInfo.CRN != nil {
		if len(*resourceInfo.CRN) > 0 {
			record = new(CloudResource)
			// for service name, look into crn and parse out the name + type (if needed)
			resourceCRN, resourceCRNErr := crn.Parse(*resourceInfo.CRN)
			if resourceCRNErr != nil {
				return nil, fmt.Errorf("error parsing CRN (%s) for data loading: %w", *resourceInfo.CRN, resourceCRNErr)
			}
			serviceName := resourceCRN.ServiceName
			if len(resourceCRN.ResourceType) > 0 {
				serviceName = serviceName + "." + resourceCRN.ResourceType
			}

			record.ServiceName = serviceName
			record.Region = "none" // assume none until we know it was valid value
			record.Count = 1

			// we assumed a none for region, now set true region if it was set
			if resourceInfo.RegionID != nil {
				if len(*resourceInfo.RegionID) > 0 {
					record.Region = *resourceInfo.RegionID
				}
			}
		}
	}

	// record may be nil at this point, which means we want to "skip" this record
	return record, nil
}

// this helper function will update resource data in the table.
// if the resource already exists, the count will be increased.
// if the resource is new, a new record is created with count 1.
func (svc *TestSetupService) updateResourceData(resource *CloudResource) error {

	// Create read-only transaction
	txn := svc.ResourceDB.Txn(false)
	defer txn.Abort()

	// look up in db, if already there bump count, else insert new row
	existing, existingErr := txn.First("resource", "id", resource.ServiceName, resource.Region)
	if existingErr != nil {
		return existingErr
	}

	// create write transaction
	writeTxn := svc.ResourceDB.Txn(true)
	defer writeTxn.Abort()

	// exists, bump the count
	if existing != nil {
		// do not use the existing object (which is how to update). Doing this based on official docs:
		// see: https://pkg.go.dev/github.com/hashicorp/go-memdb#Txn.Insert
		updateResource := existing.(*CloudResource).Copy()
		updateResource.Count++
		updateErr := writeTxn.Insert("resource", updateResource)
		if updateErr != nil {
			return fmt.Errorf("error during resource record UPDATE: %w", updateErr)
		}
	} else {
		// INSERT new service record
		resource.Count = 1
		insertErr := writeTxn.Insert("resource", resource)
		if insertErr != nil {
			return fmt.Errorf("error during resource record INSERT: %w", insertErr)
		}
	}

	writeTxn.Commit()

	return nil
}

// this helper function will create service limit records in the table.
func (svc *TestSetupService) insertCloudServiceLimit(limit *CloudServiceLimit) error {

	// create write transaction
	writeTxn := svc.ResourceDB.Txn(true)
	defer writeTxn.Abort()

	insertErr := writeTxn.Insert("limit", limit)
	if insertErr != nil {
		return fmt.Errorf("error during service limit record INSERT for service name %s: %w", limit.ServiceName, insertErr)
	}

	writeTxn.Commit()

	return nil
}

func (svc *TestSetupService) GetServiceRegionLimitation(serviceName string) (*CloudServiceLimit, error) {

	txn := svc.ResourceDB.Txn(false)
	defer txn.Abort()

	limit, getErr := txn.First("limit", "id", serviceName)
	if getErr != nil {
		return nil, getErr
	}

	return limit.(*CloudServiceLimit), nil
}

func (svc *TestSetupService) PrintResourceTableData() {
	read := svc.ResourceDB.Txn(false)
	defer read.Abort()

	records, readErr := read.Get("resource", "id")
	if readErr != nil {
		fmt.Printf("ERROR printing DB records: %s", readErr)
		return
	}

	fmt.Println("========== START OF TEST SETUP RESOURCE DUMP =============")
	for obj := records.Next(); obj != nil; obj = records.Next() {
		resource := obj.(*CloudResource)
		fmt.Println(resource)
	}
	fmt.Println("========== END OF TEST SETUP RESOURCE DUMP =============")
}

func (svc *TestSetupService) PrintLimitsTableData() {
	read := svc.ResourceDB.Txn(false)
	defer read.Abort()

	records, readErr := read.Get("limit", "id")
	if readErr != nil {
		fmt.Printf("ERROR printing limit records: %s", readErr)
		return
	}

	fmt.Println("========== START OF TEST SETUP LIMIT DUMP =============")
	for obj := records.Next(); obj != nil; obj = records.Next() {
		limit := obj.(*CloudServiceLimit)
		fmt.Println(limit)
	}
	fmt.Println("========== END OF TEST SETUP LIMIT DUMP =============")
}
