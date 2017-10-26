pragma solidity ^0.4.8;

contract ERC20 {
    // traditional token balance
    mapping (address => uint256) balances;
    
    // privacy balance, bytes for public key 
    mapping (address => uint256) public privacyBalance;
    mapping (address => bytes) public otaKey;
         
    //this only for initialize, only for test to mint token to one wan address
    function initPrivacyAsset(address initialBase, bytes baseKeyBytes, uint256 value) {
        privacyBalance[initialBase] = value;
        otaKey[initialBase] = baseKeyBytes;
    }   
    
    // return string just for debug
    function otatransfer(address _to, bytes _toKey, uint256 _value) returns (string) {      
        if(privacyBalance[msg.sender] < _value) return "sender token too low";
        
        privacyBalance[msg.sender] -= _value;
        privacyBalance[_to] += _value;
        otaKey[_to] = _toKey;
        return "success";
    }   
    
    function transfer(address _to, uint256 _value) returns (bool success) {
        if (balances[msg.sender] >= _value && balances[_to] + _value > balances[_to]) {
            balances[msg.sender] -= _value;
            balances[_to] += _value;
            return true;
        } else { return false; }
    }

    function transferFrom(address _from, address _to, uint256 _value) returns (bool success) {
        if (balances[_from] >= _value && balances[_to] + _value > balances[_to]) {
            balances[_to] += _value;
            balances[_from] -= _value;
            return true;
        } else { return false; }
    }

    function balanceOf(address _owner) constant returns (uint256 balance) {
        return balances[_owner];
    }

    function otabalanceOf(address _owner) constant returns (uint256 balance) {
        return privacyBalance[_owner];
    }
}


/*
bytecode = "0x6060604052341561000f57600080fd5b5b610c218061001f6000396000f3006060604052361561008c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063209194e61461009157806323b872dd1461019057806341267ca21461020957806370a0823114610256578063a3796c15146102a3578063a9059cbb14610328578063ce6ebd3d14610382578063f8a5b335146103cf575b600080fd5b341561009c57600080fd5b610114600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001908201803590602001908080601f01602080910402602001604051908101604052809392919081815260200183838082843782019150505050505091908035906020019091905050610482565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156101555780820151818401525b602081019050610139565b50505050905090810190601f1680156101825780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561019b57600080fd5b6101ef600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001909190505061063c565b604051808215151515815260200191505060405180910390f35b341561021457600080fd5b610240600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506107c0565b6040518082815260200191505060405180910390f35b341561026157600080fd5b61028d600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506107d8565b6040518082815260200191505060405180910390f35b34156102ae57600080fd5b610326600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001908201803590602001908080601f01602080910402602001604051908101604052809392919081815260200183838082843782019150505050505091908035906020019091905050610821565b005b341561033357600080fd5b610368600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919080359060200190919050506108bf565b604051808215151515815260200191505060405180910390f35b341561038d57600080fd5b6103b9600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610a42565b6040518082815260200191505060405180910390f35b34156103da57600080fd5b610406600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610a8c565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156104475780820151818401525b60208101905061042b565b50505050905090810190601f1680156104745780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b61048a610b3c565b81600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054101561050e576040805190810160405280601481526020017f73656e64657220746f6b656e20746f6f206c6f770000000000000000000000008152509050610635565b81600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254039250508190555081600160008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254019250508190555082600260008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090805190602001906105fb929190610b50565b506040805190810160405280600781526020017f737563636573730000000000000000000000000000000000000000000000000081525090505b9392505050565b6000816000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020541015801561070957506000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054826000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205401115b156107af57816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282540192505081905550816000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282540392505081905550600190506107b9565b600090506107b9565b5b9392505050565b60016020528060005260406000206000915090505481565b60008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205490505b919050565b80600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555081600260008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090805190602001906108b8929190610b50565b505b505050565b6000816000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020541015801561098c57506000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054826000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205401115b15610a3257816000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282540392505081905550816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254019250508190555060019050610a3c565b60009050610a3c565b5b92915050565b6000600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205490505b919050565b60026020528060005260406000206000915090508054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610b345780601f10610b0957610100808354040283529160200191610b34565b820191906000526020600020905b815481529060010190602001808311610b1757829003601f168201915b505050505081565b602060405190810160405280600081525090565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10610b9157805160ff1916838001178555610bbf565b82800160010185558215610bbf579182015b82811115610bbe578251825591602001919060010190610ba3565b5b509050610bcc9190610bd0565b5090565b610bf291905b80821115610bee576000816000905550600101610bd6565b5090565b905600a165627a7a723058209b5e648e3e5699fd07b8ec2c50e475c5472d844193edf7d2195cb71e1a96413a0029"
abi = "[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_toKey","type":"bytes"},{"name":"_value","type":"uint256"}],"name":"otatransfer","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"privacyBalance","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":false,"inputs":[{"name":"initialBase","type":"address"},{"name":"baseKeyBytes","type":"bytes"},{"name":"value","type":"uint256"}],"name":"initPrivacyAsset","outputs":[],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"otabalanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"otaKey","outputs":[{"name":"","type":"bytes"}],"payable":false,"type":"function","stateMutability":"view"}]"
*/