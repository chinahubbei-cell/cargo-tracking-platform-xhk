const axios = require('axios');

async function testPurchase() {
    try {
        const res = await axios.post('http://localhost:5180/api/trade/orders/purchase', {
            contact_name: "Test",
            phone: "123456789",
            items: [
                { product_name: "X3 追货版", qty: 1 }
            ]
        }, {
            headers: {
                // If there's a token, set it here, or omit to expect 401
            }
        });
        console.log("Success:", res.data);
    } catch (err) {
        if (err.response) {
            console.log("HTTP Error:", err.response.status, err.response.data);
        } else {
            console.log("NETWORK/OTHER ERROR:", err.message);
        }
    }
}

testPurchase();
