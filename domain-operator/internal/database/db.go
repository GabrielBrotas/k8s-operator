package database

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	domainv1alpha1 "github.com/gabriel-brotas/domain-operator/api/v1alpha1"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	Conn *pgxpool.Pool
)

func GetDBConnection() *pgxpool.Pool {
	if Conn != nil {
		log.Println("using existing database connection pool")
		return Conn
	}

	log.Println("opening a new database connection pool")

	connectionString := os.Getenv("DATABASE_URL")

	if connectionString == "" {
		// connectionString = "postgres://admin:postgresqladmin@postgresql.default.svc.cluster.local:5432/postgresqlDB"
		connectionString = "postgres://admin:admin@localhost:5432/postgresqlDB"
	}

	maxConnections := os.Getenv("DB_MAX_CONNECTIONS")

	if maxConnections == "" {
		maxConnections = "50"
	}

	config, err := pgxpool.ParseConfig(connectionString)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	maxConnectionsInt, err := strconv.ParseInt(maxConnections, 10, 64)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	config.MaxConns = int32(maxConnectionsInt)
	config.MinConns = int32(maxConnectionsInt)

	config.MaxConnIdleTime = time.Minute * 3

	Conn, err = pgxpool.NewWithConfig(context.Background(), config)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Println("connected to database")

	log.Println("pinging database...")

	err = Conn.Ping(context.Background())

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Println("pinged database")

	return Conn
}

func GetDomain(pool *pgxpool.Pool, domainID string) (domainv1alpha1.DomainSpec, error) {
	const sql = `SELECT domain_id, environments FROM domains WHERE domain_id = $1`
	row := pool.QueryRow(context.Background(), sql, domainID)
	var domain domainv1alpha1.DomainSpec
	err := row.Scan(&domain.DomainID, &domain.Environments)

	if err != nil && err.Error() == "no rows in result set" {
		return domain, nil
	}

	return domain, err
}

func CreateDomain(pool *pgxpool.Pool, domain domainv1alpha1.DomainSpec) error {
	const sql = `INSERT INTO domains(domain_id, environments) VALUES ($1, $2)`
	_, err := pool.Exec(context.Background(), sql, domain.DomainID, domain.Environments)
	return err
}

func UpdateDomain(pool *pgxpool.Pool, domain domainv1alpha1.DomainSpec) error {
	const sql = `UPDATE domains SET environments = $2 WHERE domain_id = $1`
	_, err := pool.Exec(context.Background(), sql, domain.DomainID, domain.Environments)
	return err
}

func DeleteDomain(pool *pgxpool.Pool, domainID string) error {
	const sql = `DELETE FROM domains WHERE domain_id = $1`
	_, err := pool.Exec(context.Background(), sql, domainID)
	return err
}
