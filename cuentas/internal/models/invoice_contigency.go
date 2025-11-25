package models

func (i *Invoice) IsInContingency() bool {
	return i.ContingencyPeriodID != nil
}

func (i *Invoice) HasSignature() bool {
	return i.DteSigned != nil && *i.DteSigned != ""
}

func (i *Invoice) NeedsSignature() bool {
	return !i.HasSignature() && i.DteUnsigned != nil
}

func (i *Invoice) IsPendingSignature() bool {
	return i.DteTransmissionStatus == "pending_signature"
}

func (i *Invoice) IsContingencyQueued() bool {
	return i.DteTransmissionStatus == "contingency_queued"
}

func (i *Invoice) IsFailedRetry() bool {
	return i.DteTransmissionStatus == "failed_retry"
}

func (i *Invoice) IsProcesado() bool {
	return i.DteTransmissionStatus == "procesado"
}

func (i *Invoice) IsRechazado() bool {
	return i.DteTransmissionStatus == "rechazado"
}
