terraform {
    required_providers  {
        test = {
            source = "hashicorp/test"
        }
        othertest = {
            source = "happycorp/test"
        }
    }
}

resource "test_instance" "exists" {
    // I exist!
} 

module "module" {
    source = "./module"
}