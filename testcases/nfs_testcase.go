package testcases

import (

	. "github.com/cloudfoundry-incubator/disaster-recovery-acceptance-tests/common"
	"os"
	"path/filepath"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/go-sql-driver/mysql"
	"time"
	"database/sql"
)

type NFSTestCase struct {
	uniqueTestID string
	instanceName string
}

func NewNFSTestCases() *NFSTestCase {
	id := RandomStringNumber()
	name := fmt.Sprintf("service-instance-%s", id)
	return &NFSTestCase{uniqueTestID: id, instanceName: name}
}

func (tc *NFSTestCase) BeforeBackup(config Config) {
	By("nfs before backup")
	cwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	fmt.Printf("-------->cwd %s", cwd)
	RunCommandSuccessfully("cf login --skip-ssl-validation -a", config.DeploymentToBackup.ApiUrl, "-u", config.DeploymentToBackup.AdminUsername, "-p", config.DeploymentToBackup.AdminPassword)
	RunCommandSuccessfully("cf create-org acceptance-test-org-" + tc.uniqueTestID)
	RunCommandSuccessfully("cf create-space acceptance-test-space-" + tc.uniqueTestID + " -o acceptance-test-org-" + tc.uniqueTestID)
	RunCommandSuccessfully("cf target -s acceptance-test-space-" + tc.uniqueTestID + " -o acceptance-test-org-" + tc.uniqueTestID)
	RunCommandSuccessfully("cf push dratsApp --docker-image docker/httpd --no-start")

	RunCommandSuccessfully("cf push " + os.Getenv("PUSHED_BROKER_NAME") + " -p " + os.Getenv("APPLICATION_PATH") + " -f " + filepath.Join(os.Getenv("APPLICATION_PATH"), "/manifest.yml"))
	RunCommandSuccessfully("cf create-service-broker " + os.Getenv("PUSHED_BROKER_NAME") + " " + os.Getenv("BROKER_USER") + " " + os.Getenv("BROKER_PASSWORD") + " " + os.Getenv("BROKER_URL"))
	RunCommandSuccessfully("cf enable-service-access " + os.Getenv("SERVICE_NAME"))
	RunCommandSuccessfully("cf create-service " + os.Getenv("SERVICE_NAME")+ " " + os.Getenv("PLAN_NAME") + " " + tc.instanceName + " -c " + fmt.Sprintf("'{\"share\":\"%s%s\"}'", os.Getenv("SERVER_ADDRESS"), os.Getenv("SHARE")))
}

func (tc *NFSTestCase) AfterBackup(config Config) {
	By("nfs after backup")
	//RunCommandSuccessfully("cf delete-service " + config.InstanceName + " -f")
	// go tamper with the db to delete its records

	dbConnectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	cfg, err := mysql.ParseDSN(dbConnectionString)
	Expect(err).NotTo(HaveOccurred())

	cfg.Timeout = 10 * time.Minute
	cfg.ReadTimeout = 10 * time.Minute
	cfg.WriteTimeout = 10 * time.Minute
	dbConnectionString = cfg.FormatDSN()

	sqlDB, err := sql.Open("mysql", dbConnectionString)
	Expect(err).NotTo(HaveOccurred())

	_, err = sqlDB.Exec(`
			DROP TABLE service_instances
	`)
	Expect(err).NotTo(HaveOccurred())

  err = sqlDB.Close()
	Expect(err).NotTo(HaveOccurred())
}

func (tc *NFSTestCase) AfterRestore(config Config) {
	By("nfs after backup")
	RunCommandSuccessfully("cf bind-service dratsApp " + tc.instanceName + ` -c '{"uid":5000,"gid":5000}'`)
}

func (tc *NFSTestCase) Cleanup(config Config) {
	By("nfs cleanup")
	RunCommandSuccessfully("cf delete-org -f acceptance-test-org-" + tc.uniqueTestID)
	RunCommandSuccessfully("cf delete-service-broker -f " + os.Getenv("PUSHED_BROKER_NAME"))
}

func (tc *NFSTestCase) deletePushedApps(config Config) {
}
