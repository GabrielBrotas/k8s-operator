package database

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	platformv1 "github.com/example/team-operator/api/v1"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	Conn *pgxpool.Pool
)

func GetDBConnection() *pgxpool.Pool {
	if Conn != nil {
		log.Println("using existing connection")
		return Conn
	}

	log.Println("opening connections")

	connectionString := os.Getenv("DATABASE_URL")
	maxConnections := os.Getenv("DB_MAX_CONNECTIONS")

	if maxConnections == "" {
		maxConnections = "50"
	}

	config, err := pgxpool.ParseConfig(connectionString)

	if err != nil {
		log.Fatal(err)
	}

	maxConnectionsInt, err := strconv.ParseInt(maxConnections, 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	config.MaxConns = int32(maxConnectionsInt)
	config.MinConns = int32(maxConnectionsInt)

	config.MaxConnIdleTime = time.Minute * 3

	Conn, err = pgxpool.NewWithConfig(context.Background(), config)

	if err != nil {
		log.Fatal(err)
	}

	err = Conn.Ping(context.Background())

	if err != nil {
		log.Fatal(err)
	}

	return Conn
}

func CreateTeam(pool *pgxpool.Pool, team platformv1.TeamSpec) error {
	const sql = `INSERT INTO teams(team_id, team_name, k8s_cluster_name, artifactory_namespace) VALUES ($1, $2, $3, $4)`
	_, err := pool.Exec(context.Background(), sql, team.TeamID, team.TeamName, team.K8sClusterName, team.ArtifactoryNamespace)
	return err
}
