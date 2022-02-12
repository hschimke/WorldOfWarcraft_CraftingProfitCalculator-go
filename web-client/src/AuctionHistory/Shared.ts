import React, { Dispatch } from 'react';

const AuctionHistoryDispatch = React.createContext<Dispatch<any> | undefined>(undefined);

export {AuctionHistoryDispatch};