module.exports = {
  extend: (web3) => {
    function insertMethod(name, call, params, inputFormatter, outputFormatter) {
      return new web3._extend.Method({ name, call, params, inputFormatter, outputFormatter });
    }

    function insertProperty(name, getter, outputFormatter) {
      return new web3._extend.Property({ name, getter, outputFormatter });
    }

    //POS
    web3._extend({
      property: 'pos',
      methods: [
        new web3._extend.Method({
          name: 'getMaxStableBlkNumber',
          call: 'pos_getMaxStableBlkNumber',
          params: 0
        }),
      ]
    });
  },
};
