export type ChargeReq = {
  id: string;
  amount: number;
  customerId: string;
  vip?: boolean;
};

export type ChargeResult = {
  id: string;
  amount: number;
  captured: number;
  discount?: number;
};
