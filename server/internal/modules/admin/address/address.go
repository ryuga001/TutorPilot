package address

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tutorpilot/internal/pkg/pg"
)

var ErrNotFound = errors.New("address not found")

type Input struct {
	LocalAddress string `json:"local_address"`
	City         string `json:"city"`
	State        string `json:"state"`
	Country      string `json:"country"`
}

type Address struct {
	ID           int
	CustomerID   int
	LocalAddress string
	City         string
	State        string
	Country      string
}

type View struct {
	ID           int    `json:"id"`
	LocalAddress string `json:"local_address"`
	City         string `json:"city"`
	State        string `json:"state"`
	Country      string `json:"country"`
}

func (a *Address) View() *View {
	return &View{
		ID: a.ID, LocalAddress: a.LocalAddress, City: a.City, State: a.State, Country: a.Country,
	}
}

const cols = `id, customer_id, local_address, city, state, country`

func scan(row pgx.Row) (*Address, error) {
	a := &Address{}
	err := row.Scan(&a.ID, &a.CustomerID, &a.LocalAddress, &a.City, &a.State, &a.Country)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func Create(ctx context.Context, q pg.Querier, customerID int, in Input) (int, error) {
	var id int
	err := q.QueryRow(ctx,
		`INSERT INTO addresses (customer_id, local_address, city, state, country)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		customerID, in.LocalAddress, in.City, in.State, in.Country,
	).Scan(&id)
	return id, err
}

func Update(ctx context.Context, q pg.Querier, customerID, id int, in Input) error {
	tag, err := q.Exec(ctx,
		`UPDATE addresses SET local_address = $3, city = $4, state = $5, country = $6, updated_at = now()
		 WHERE customer_id = $1 AND id = $2`,
		customerID, id, in.LocalAddress, in.City, in.State, in.Country)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func Get(ctx context.Context, q pg.Querier, customerID, id int) (*Address, error) {
	return scan(q.QueryRow(ctx, `SELECT `+cols+` FROM addresses WHERE customer_id = $1 AND id = $2`, customerID, id))
}

func Delete(ctx context.Context, q pg.Querier, customerID, id int) error {
	_, err := q.Exec(ctx, `DELETE FROM addresses WHERE customer_id = $1 AND id = $2`, customerID, id)
	return err
}
