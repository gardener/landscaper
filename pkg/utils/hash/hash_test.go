package hash

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hash computation tests", func() {

	Context("Elementary types", func() {

		It("should hash strings", func() {
			result, err := ComputeHash("test")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"))

			result, err = ComputeHash("hello")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"))
		})

		It("should hash booleans", func() {
			result, err := ComputeHash(true)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b"))

			result, err = ComputeHash(false)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))
		})

		It("should hash integers", func() {
			result, err := ComputeHash(1)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"))

			result, err = ComputeHash(42)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"))
		})
	})

	Context("Initial values", func() {

		It("should hash initial values", func() {
			result, err := ComputeHash("")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))

			result, err = ComputeHash(0)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))

			result, err = ComputeHash(false)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))

			result, err = ComputeHash([]interface{}{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))

			result, err = ComputeHash(map[string]interface{}{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))

			result, err = ComputeHash(map[string]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))

			result, err = ComputeHash(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("426b5dfece37e413f559015825ebc7c5ba251a13028e2fcd5ed36df57be00b6c"))
		})

		Context("Maps", func() {

			It("should hash maps", func() {
				m := map[string]string{"a": "1", "b": "2"}

				result, err := ComputeHash(m)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("c6879d189ada98c7f4acce4fb0098741c582a53468b4387e2759c69c85845bcf"))

				m = map[string]string{"b": "2", "a": "1"}

				result, err = ComputeHash(m)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("c6879d189ada98c7f4acce4fb0098741c582a53468b4387e2759c69c85845bcf"))

				s := []string{"b", "2", "a", "1"}

				result, err = ComputeHash(s)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("c6879d189ada98c7f4acce4fb0098741c582a53468b4387e2759c69c85845bcf"))
			})
		})

		Context("Slices", func() {

			It("should hash maps", func() {
				s := []string{"a", "b"}

				result, err := ComputeHash(s)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("62af5c3cb8da3e4f25061e829ebeea5c7513c54949115b1acc225930a90154da"))

				s = []string{"b", "a"}

				result, err = ComputeHash(s)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("ab19ec537f09499b26f0f62eed7aefad46ab9f498e06a7328ce8e8ef90da6d86"))

				ha, err := ComputeHash("a")
				Expect(err).NotTo(HaveOccurred())
				Expect(ha).To(Equal("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"))

				hha, err := ComputeHash(ha)
				Expect(err).NotTo(HaveOccurred())
				Expect(hha).To(Equal("da3811154d59c4267077ddd8bb768fa9b06399c486e1fc00485116b57c9872f5"))

				s = []string{"a"}

				result, err = ComputeHash(s)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("da3811154d59c4267077ddd8bb768fa9b06399c486e1fc00485116b57c9872f5"))
			})
		})
	})
})
