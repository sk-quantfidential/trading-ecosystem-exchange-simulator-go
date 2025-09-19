package services

import (
	"github.com/sirupsen/logrus"
)

type ExchangeService struct {
	logger *logrus.Logger
}

func NewExchangeService(logger *logrus.Logger) *ExchangeService {
	return &ExchangeService{
		logger: logger,
	}
}

func (s *ExchangeService) PlaceOrder(symbol string, quantity float64, price float64, side string) (string, error) {
	s.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"quantity": quantity,
		"price":    price,
		"side":     side,
	}).Info("Placing order")
	return "order-123", nil
}

func (s *ExchangeService) GetOrderStatus(orderID string) (string, error) {
	s.logger.WithField("orderID", orderID).Info("Getting order status")
	return "filled", nil
}