axios.defaults.xsrfCookieName = "X-CSRF-Token"
axios.defaults.xsrfHeaderName = "X-CSRF-Token"

const transport = axios.create({
    timeout: 10000,
    withCredentials: true
});

Vue.component('serviceaccount-add-widget', {
    props: ['orgid'],
    data: function () {
        return {
            accountName: "Default Service Account"
        }
    },
    methods: {
        addServiceAccount: function (event) {
            let that = this;
            transport.post('/api/gcp/service-account', {
                organizationId: that.orgid,
                name: that.accountName
            })
                .then(function (response) {
                    alert('Added ' + that.accountName);
                    console.log(response);
                })
                .catch(function (error) {
                    console.log(error);
                });
        }
    },
    template: `
    <div>
        <input type="text" name="accountName" v-model="accountName"></input>
        <button name="addServiceAccount" v-on:click="addServiceAccount">Add Service Account</button>
    </div>
    `
})


var userManagementView = new Vue({
    el: '#user-organization-view',
    delimiters: ['${', '}'],
    data: {
    },
    methods: {
    }
})
