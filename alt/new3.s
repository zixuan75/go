TEXT	main(SB),512|7,$0
	ADD	R1, R2, R3
	ADD $42, R1, R3
	SLLD	R1, R2, R3
	SLLD	$2, R1, R3
	FCMPD	D2, D4
	FCMPS	F1, F2
	CMP	R1, R2
	RET