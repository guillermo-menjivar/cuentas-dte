✅ precioUni = Price WITH IVA (what customer sees)
✅ ventaGravada = precioUni × cantidad - montoDescu (ALSO with IVA!)
✅ ivaItem = ventaGravada - (ventaGravada / 1.13)
✅ subTotal = totalGravada (no separate addition of IVA!)
✅ totalPagar = subTotal (everything already includes IVA)


```javascript
// Customer pays total WITH IVA
const totalWithIVA = 11.30;

// Item level
const precioUni = totalWithIVA;
const ventaGravada = precioUni; // Same!
const ivaItem = ventaGravada - (ventaGravada / 1.13); // Extract IVA

// Resumen level
const totalGravada = ventaGravada; // WITH IVA
const totalIva = ivaItem; // Extracted IVA
const subTotal = totalGravada; // NOT totalGravada + totalIva!
const totalPagar = subTotal;
```
