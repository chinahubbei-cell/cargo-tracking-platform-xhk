async function testPurchase() {
    try {
        const res = await fetch('http://localhost:5180/api/trade/orders/purchase', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                contact_name: "Test",
                phone: "123456789",
                items: [
                    { product_name: "X3 追货版", qty: 1 }
                ]
            })
        });
        const text = await res.text();
        console.log("Response Status:", res.status);
        console.log("Response Body:", text);
    } catch (err) {
        console.log("NETWORK/OTHER ERROR:", err.message);
    }
}

testPurchase();
